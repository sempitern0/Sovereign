// Package prerequisites prepares supported Linux package-management prerequisites.
// Unlike read-only plan/status, Prepare may modify the host, but only after the
// caller authorizes setup. Commands are fixed arguments, never shell strings.
package prerequisites

import (
 "context"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "runtime"
 "strings"
 "time"

 "github.com/sempitern0/Sovereign/internal/install"
)

const FlathubURL = "https://dl.flathub.org/repo/flathub.flatpakrepo"
const flathubRepo = "https://dl.flathub.org/repo"

// System isolates the host boundary for testing. Production code only uses
// approved distro package managers and resolves programs using the OS PATH.
type System interface {
 LookPath(program string) (string,error)
 ReadOSRelease() ([]byte,error)
 Root() bool
}

type HostSystem struct{}
func (HostSystem) LookPath(name string)(string,error){return exec.LookPath(name)}
func (HostSystem) ReadOSRelease()([]byte,error){return os.ReadFile("/etc/os-release")}
func (HostSystem) Root()bool{return os.Geteuid()==0}

// Prepare installs Flatpak when missing and adds a user-scoped official Flathub
// remote if necessary. It never makes changes without explicit consent.
// A noninteractive non-root run cannot issue sudo or assume elevation.
func Prepare(ctx context.Context, sys System, runner install.Runner, interactive bool, consent bool) (bool,error){
 if runtime.GOOS!="linux" {return false,nil}
 if runner==nil || sys==nil{return false,errors.New("missing prerequisite host adapter")}
 _,flatpakErr:=sys.LookPath("flatpak")
 if flatpakErr!=nil {
  if !consent{return false,errors.New("Flatpak is missing; review the plan and authorize setup with --yes")}
  data,err:=sys.ReadOSRelease()
  if err!=nil{return false,fmt.Errorf("cannot identify Linux distribution: %w",err)}
  program,args,err:=installCommand(data,sys)
  if err!=nil{return false,err}
  if !sys.Root()&&!interactive{return false,errors.New("Flatpak requires administrator approval; run setup from an interactive terminal")}
  // apt-get requires refreshed package indices on fresh Debian/Ubuntu machines.
  if program=="apt-get" || (program=="sudo" && len(args)>0 && args[0]=="apt-get") {
   updateProgram:=program
   updateArgs:=[]string{"update"}
   if program=="sudo" {updateArgs=[]string{"apt-get","update"}}
   if err:=runInstall(ctx,runner,interactive,updateProgram,updateArgs...);err!=nil{
    return false,fmt.Errorf("refresh official package indices: %w",err)
   }
  }
  if err:=runInstall(ctx,runner,interactive,program,args...);err!=nil{return false,fmt.Errorf("install Flatpak: %w",err)}
  if _,err:=sys.LookPath("flatpak");err!=nil{return false,errors.New("Flatpak installation finished but the executable is still unavailable")}
 }
 if !consent{return false,nil} // Read-only mode never adds repositories.
 remotesCtx,cancel:=context.WithTimeout(ctx,2*time.Minute)
 out,err:=runner.Run(remotesCtx,"flatpak","remotes","--user","--columns=name,url")
 cancel()
 if err!=nil{return false,fmt.Errorf("cannot inspect user Flatpak remotes: %w",err)}
 for _,line:=range strings.Split(out,"\n"){
  fields:=strings.Fields(line)
  if len(fields)>0 && fields[0]=="flathub"{
   if len(fields)==2 && strings.TrimRight(fields[1],"/")==flathubRepo{return false,nil}
   return false,errors.New("an existing Flathub remote has a different URL; refusing to overwrite it")
  }
 }
 if err:=runInstall(ctx,runner,interactive,"flatpak","remote-add","--user","--if-not-exists","flathub",FlathubURL);err!=nil{
  return false,fmt.Errorf("configure official Flathub user remote: %w",err)
 }
 verifyCtx,done:=context.WithTimeout(ctx,2*time.Minute)
 verified,err:=runner.Run(verifyCtx,"flatpak","remotes","--user","--columns=name,url")
 done()
 if err!=nil{return false,fmt.Errorf("cannot verify Flathub remote: %w",err)}
 for _,line:=range strings.Split(verified,"\n"){
  fields:=strings.Fields(line)
  if len(fields)==2 && fields[0]=="flathub" && strings.TrimRight(fields[1],"/")==flathubRepo{return true,nil}
 }
 return false,errors.New("Flathub remote was not verified after configuration")
}

func runInstall(ctx context.Context, runner install.Runner, interactive bool, program string,args ...string)error{
 installCtx,cancel:=context.WithTimeout(ctx,30*time.Minute)
 defer cancel()
 if interactive{
  if r,ok:=runner.(install.InteractiveRunner);ok{
   return r.RunInteractive(installCtx,program,args...)
  }
 }
 _,err:=runner.Run(installCtx,program,args...)
 return err
}

func parseOSRelease(data []byte)map[string]string{
 result:=map[string]string{}
 for _,line:=range strings.Split(string(data),"\n"){
  line=strings.TrimSpace(line)
  if line==""||strings.HasPrefix(line,"#"){continue}
  pair:=strings.SplitN(line,"=",2)
  if len(pair)!=2{continue}
  key:=strings.TrimSpace(pair[0])
  value:=strings.TrimSpace(pair[1])
  value=strings.Trim(value,"\"'")
  if key=="ID"||key=="ID_LIKE"{result[key]=strings.ToLower(value)}
 }
 return result
}

// Only known package managers and literal arguments are allowed; do not guess
// for unknown distributions or fall back to remote scripts.
func installCommand(release []byte,sys System)(string,[]string,error){
 fields:=parseOSRelease(release)
 ids:=append([]string{fields["ID"]},strings.Fields(fields["ID_LIKE"])...)
 for _,id:=range ids{
  var pm string
  var args []string
  switch id{
  case "ubuntu","debian","linuxmint","pop","elementary","kali","raspbian":
   pm,args="apt-get",[]string{"install","-y","flatpak"}
  case "fedora","rhel","centos","rocky","almalinux","nobara":
   pm,args="dnf",[]string{"install","-y","flatpak"}
  case "arch","manjaro","endeavouros":
   pm,args="pacman",[]string{"-S","--needed","--noconfirm","flatpak"}
  case "opensuse","opensuse-tumbleweed","opensuse-leap","suse":
   pm,args="zypper",[]string{"--non-interactive","install","flatpak"}
  default:continue
  }
  if _,err:=sys.LookPath(pm);err!=nil{
   return "",nil,fmt.Errorf("Linux distribution expects %s, but that package manager is missing",pm)
  }
  if sys.Root(){return pm,args,nil}
  if _,err:=sys.LookPath("sudo");err!=nil{return "",nil,errors.New("sudo is required to install Flatpak on this distribution")}
  return "sudo",append([]string{pm},args...),nil
 }
 return "",nil,fmt.Errorf("unsupported Linux distribution %q for automatic Flatpak installation; no system changes made",fields["ID"])
}
