package prerequisites

import (
 "context"
 "errors"
 "reflect"
 "runtime"
 "strings"
 "testing"
)

type fakeSystem struct{
 available map[string]bool
 release string
 root bool
}
func(f *fakeSystem)LookPath(name string)(string,error){
 if f.available[name]{return "/usr/bin/"+name,nil}
 return "",errors.New("not found")
}
func(f *fakeSystem)ReadOSRelease()([]byte,error){return []byte(f.release),nil}
func(f *fakeSystem)Root()bool{return f.root}
type fakeRunner struct{
 sys *fakeSystem
 remote string
 calls []string
 fail string
}
func(f *fakeRunner)Run(_ context.Context,name string,args ...string)(string,error){
 f.calls=append(f.calls,strings.Join(append([]string{name},args...)," "))
 if name==f.fail{return "",errors.New("simulated failure")}
 if name=="flatpak"&&args[0]=="remotes"{return f.remote,nil}
 if (name=="sudo"||name=="apt-get"||name=="dnf"||name=="pacman"||name=="zypper") {f.sys.available["flatpak"]=true}
 if name=="flatpak"&&args[0]=="remote-add"{f.remote="flathub\thttps://dl.flathub.org/repo/\n"}
 return "",nil
}
func TestInstallCommandSelection(t *testing.T){
 cases:=[]struct{release,manager string; args []string}{
  {"ID=ubuntu\n","sudo",[]string{"apt-get","install","-y","flatpak"}},
  {"ID=fedora\n","sudo",[]string{"dnf","install","-y","flatpak"}},
  {"ID=manjaro\n","sudo",[]string{"pacman","-S","--needed","--noconfirm","flatpak"}},
  {"ID=opensuse-tumbleweed\n","sudo",[]string{"zypper","--non-interactive","install","flatpak"}},
  {"ID=pop\nID_LIKE=\"ubuntu debian\"","sudo",[]string{"apt-get","install","-y","flatpak"}},
 }
 for _,tt:=range cases{
  sys:=&fakeSystem{available:map[string]bool{"apt-get":true,"dnf":true,"pacman":true,"zypper":true,"sudo":true},release:tt.release}
  got,args,err:=installCommand([]byte(tt.release),sys)
  if err!=nil||got!=tt.manager||!reflect.DeepEqual(args,tt.args){t.Errorf("%s got %s %v err=%v",tt.release,got,args,err)}
 }
}
func TestUnsupportedDistroFailsClosed(t *testing.T){
 sys:=&fakeSystem{available:map[string]bool{"sudo":true},release:"ID=unknown"}
 if _,_,err:=installCommand([]byte(sys.release),sys);err==nil {t.Fatal("unknown distro accepted")}
}
func TestPrepareNeverMutatesWithoutConsent(t *testing.T){
 sys:=&fakeSystem{available:map[string]bool{"apt-get":true,"sudo":true},release:"ID=ubuntu"}
 runner:=&fakeRunner{sys:sys}
 _,err:=Prepare(context.Background(),sys,runner,true,false)
 if runtime.GOOS=="linux"&&err==nil{t.Fatal("missing Flatpak accepted without consent")}
 if len(runner.calls)!=0{t.Fatalf("mutated or inspected provider: %v",runner.calls)}
}
func TestPrepareInstallsFlatpakAndFlathubIdempotently(t *testing.T){
 if runtime.GOOS!="linux"{t.Skip("Linux only")}
 sys:=&fakeSystem{available:map[string]bool{"apt-get":true,"sudo":true},release:"ID=ubuntu"}
 runner:=&fakeRunner{sys:sys}
 changed,err:=Prepare(context.Background(),sys,runner,true,true)
 if err!=nil||!changed||!sys.available["flatpak"] {t.Fatalf("%v %v",changed,err)}
 before:=len(runner.calls)
 changed,err=Prepare(context.Background(),sys,runner,true,true)
 if err!=nil||changed||len(runner.calls)!=before+1{t.Fatalf("second run: changed=%v err=%v calls=%v",changed,err,runner.calls)}
}
func TestPrepareRejectsUnsafeRemote(t *testing.T){
 if runtime.GOOS!="linux"{t.Skip("Linux only")}
 sys:=&fakeSystem{available:map[string]bool{"flatpak":true}}
 runner:=&fakeRunner{sys:sys,remote:"flathub https://evil.example/repo"}
 if _,err:=Prepare(context.Background(),sys,runner,true,true);err==nil{t.Fatal("unsafe remote accepted")}
 if len(runner.calls)!=1{t.Fatalf("unexpected changes %v",runner.calls)}
}
func TestNoSudoInUnattendedMode(t *testing.T){
 if runtime.GOOS!="linux"{t.Skip("Linux only")}
 sys:=&fakeSystem{available:map[string]bool{"sudo":true,"apt-get":true},release:"ID=debian"}
 runner:=&fakeRunner{sys:sys}
 if _,err:=Prepare(context.Background(),sys,runner,false,true);err==nil{t.Fatal("unattended elevated install accepted")}
 if len(runner.calls)!=0{t.Fatalf("ran package manager: %v",runner.calls)}
}
