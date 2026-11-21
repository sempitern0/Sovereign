package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
)

// A dependency-free text wizard; installation is always a separate confirmed step.
type wizard struct {
	input   *bufio.Scanner
	output  io.Writer
	execute func(...string) int
	catalog catalog.Catalog
}

func newWizard(in io.Reader, out io.Writer, execute func(...string) int, c catalog.Catalog) *wizard {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 256), 4096)
	return &wizard{scanner, out, execute, c}
}
func (w *wizard) ask(prompt string) (string, bool) {
	fmt.Fprint(w.output, prompt)
	if !w.input.Scan() {
		return "", false
	}
	return strings.TrimSpace(w.input.Text()), true
}
func (w *wizard) choose(max int) (int, bool) {
	for {
		answer, ok := w.ask("\nOpción: ")
		if !ok {
			return 0, false
		}
		n, err := strconv.Atoi(answer)
		if err == nil && n >= 0 && n <= max {
			return n, true
		}
		fmt.Fprintf(w.output, "Elige un número del 0 al %d.\n", max)
	}
}
func (w *wizard) confirm() bool {
	answer, ok := w.ask("Para autorizar la instalación escribe SI: ")
	return ok && strings.EqualFold(answer, "SI")
}

// prepare follows one stable installation definition. No profile selection or
// machine modification occurs before the user reviews the local plan.
func (w *wizard) prepare() {
	if w.execute("plan", "--profile", "essential") != 0 {
		fmt.Fprintln(w.output, "No se pudo consultar el estado. No se ha instalado nada.")
		return
	}
	answer, ok := w.ask("\n1) Instalar lo que falte  2) Volver sin cambios\nOpción: ")
	if !ok || answer != "1" {
		return
	}
	if !w.confirm() {
		fmt.Fprintln(w.output, "Cancelado. No se ha autorizado ninguna instalación.")
		return
	}
	if w.execute("setup", "--yes") != 0 {
		fmt.Fprintln(w.output, "La preparación no se completó. Revisa los errores para reintentar.")
	}
}
func (w *wizard) apps() {
	engine := install.Engine{Platform: runtime.GOOS}
	var ids []string
	for _, id := range w.catalog.AppIDs() {
		if _, err := engine.Resolve(w.catalog, id, runtime.GOOS); err == nil {
			ids = append(ids, id)
		}
	}
	fmt.Fprintln(w.output, "\nAplicaciones aprobadas")
	for i, id := range ids {
		fmt.Fprintf(w.output, " %d) %s\n", i+1, w.catalog.Apps[id].Name)
	}
	fmt.Fprintln(w.output, " 0) Volver")
	n, ok := w.choose(len(ids))
	if !ok || n == 0 {
		return
	}
	id := ids[n-1]
	target, err := engine.Resolve(w.catalog, id, runtime.GOOS)
	if err != nil {
		fmt.Fprintln(w.output, "Paquete no autorizado.")
		return
	}
	fmt.Fprintf(w.output, "%s — %s (%s); alcance: %s; origen: %s\n",
		w.catalog.Apps[id].Name, target.Provider, target.PackageID, target.Scope, target.Provenance)
	if target.NeedsAdminApproval() {
		fmt.Fprintln(w.output, "Aviso: puede solicitar privilegios de administrador.")
	}
	if target.Provenance == "community" {
		fmt.Fprintln(w.output, "Aviso: paquete mantenido por la comunidad.")
	}
	for _, dep := range target.ProviderDependencies {
		fmt.Fprintf(w.output, "Dependencia declarada: %s\n", dep)
	}
	state, err := (install.Engine{Runner: newRunner(), Platform: runtime.GOOS}).Status(context.Background(), target)
	if err != nil {
		fmt.Fprintf(w.output, "No se pudo verificar el paquete: %v\n", err)
		return
	}
	if state == install.Installed {
		fmt.Fprintln(w.output, "Ya está instalado. Sin cambios.")
		return
	}
	if !w.confirm() {
		fmt.Fprintln(w.output, "Cancelado. Sin instalaciones autorizadas.")
		return
	}
	if w.execute("apply", "--app", id, "--yes") != 0 {
		fmt.Fprintln(w.output, "Instalación no completada.")
	}
}
func (w *wizard) run() int {
	fmt.Fprintln(w.output, "SOVEREIGN — Preparar y disfrutar")
	fmt.Fprintln(w.output, "Local-first | Multisuite opcional")
	for {
		fmt.Fprintln(w.output, "\n1) Preparar este equipo")
		fmt.Fprintln(w.output, "2) Comprobar el entorno")
		fmt.Fprintln(w.output, "3) Aplicaciones adicionales")
		fmt.Fprintln(w.output, "4) Explorar catálogo")
		fmt.Fprintln(w.output, "5) Diagnóstico del equipo")
		fmt.Fprintln(w.output, "6) Opciones avanzadas")
		fmt.Fprintln(w.output, "7) Privacidad de Windows (auditoría)")
		fmt.Fprintln(w.output, "8) Mapas, Wikipedia y biblioteca offline")
		fmt.Fprintln(w.output, "0) Salir")
		n, ok := w.choose(8)
		if !ok || n == 0 {
			fmt.Fprintln(w.output, "Hasta pronto.")
			return 0
		}
		switch n {
		case 1:
			w.prepare()
		case 2:
			w.execute("status")
		case 3:
			w.apps()
		case 4:
			w.execute("catalog", "list", "--platform", runtime.GOOS)
		case 5:
			w.execute("doctor")
		case 6:
			fmt.Fprintln(w.output, "Comandos avanzados desde PowerShell o bash:")
			fmt.Fprintln(w.output, "  sovereign status --json")
			fmt.Fprintln(w.output, "  sovereign plan --profile essential --verbose")
			fmt.Fprintln(w.output, "  sovereign setup --yes")
			fmt.Fprintln(w.output, "  sovereign status --verbose")
		case 7:
			if runtime.GOOS == "windows" {
				w.execute("privacy", "audit")
			} else {
				fmt.Fprintln(w.output, "La auditoría de privacidad está disponible únicamente en Windows.")
			}
		case 8:
			fmt.Fprintln(w.output, "Biblioteca offline — instalación y datos separados:")
			fmt.Fprintln(w.output, "  Kiwix: Wikipedia y otros contenidos ZIM — https://library.kiwix.org/")
			fmt.Fprintln(w.output, "  CoMaps: mapas regionales que deben descargarse antes de viajar — https://www.comaps.app/")
			fmt.Fprintln(w.output, "  Las aplicaciones y datos aún no están habilitados en el instalador de Sovereign.")
			fmt.Fprintln(w.output, "  Documentación: docs/OFFLINE_LIBRARY.md")
		}
	}
}
func runWizard() int {
	if !isTerminal() {
		fmt.Fprintln(os.Stderr, "El asistente requiere una terminal interactiva. Usa status, plan o apply para automatizaciones.")
		return 2
	}
	c, err := loadCatalog("")
	if err != nil {
		return loadError(err)
	}
	return newWizard(os.Stdin, os.Stdout, func(args ...string) int { return run(args) }, c).run()
}
