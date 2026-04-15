package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/gen"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Code generators (module, CRUD, wire)",
}

var genModuleCmd = &cobra.Command{
	Use:   "module [name]",
	Short: "Generate handler/service/repository/register under modules/<name>/",
	Long: `Creates a small vertical slice:

  modules/<name>/{repository,service,handler,register}.go

By default, patches the wire file (internal/routes/register.go or pkg/server/register_modules.go)
with the import and Register() call. Use --skip-wire to only generate files.`,
	Args: cobra.ExactArgs(1),
	RunE: runGenModule,
}

var genCrudCmd = &cobra.Command{
	Use:   "crud [name]",
	Short: "Add REST CRUD stubs (crud_handlers.go + crud_register.go) under modules/<name>/",
	Long: `Adds crud_handlers.go and crud_register.go under an existing modules/<name>/ package.

By default, patches the wire file with RegisterCRUD(). Use --skip-wire to skip.`,
	Args: cobra.ExactArgs(1),
	RunE: runGenCrud,
}

var genWireCmd = &cobra.Command{
	Use:   "wire [name]",
	Short: "Patch wire markers only (no code generation)",
	Long: `Updates internal/routes/register.go or pkg/server/register_modules.go from modules/<name>/.

By default, adds Register() if register.go exists and RegisterCRUD() if crud_register.go exists.
Use this after --skip-wire, or to re-apply wiring without overwriting module files.

Runs "go fmt" on the wire file when the Go toolchain is available.`,
	Args: cobra.ExactArgs(1),
	RunE: runGenWire,
}

func init() {
	genModuleCmd.Flags().String("out", "modules", "base directory for generated modules (relative to module-root)")
	genModuleCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genModuleCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genModuleCmd.Flags().Bool("skip-wire", false, "do not patch wire markers after generate")
	genCrudCmd.Flags().String("out", "modules", "base directory (relative to module-root)")
	genCrudCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genCrudCmd.Flags().Bool("dry-run", false, "print paths only, do not write files")
	genCrudCmd.Flags().Bool("skip-wire", false, "do not patch wire markers after generate")
	genWireCmd.Flags().String("out", "modules", "base directory (relative to module-root)")
	genWireCmd.Flags().String("module-root", ".", "directory containing go.mod")
	genWireCmd.Flags().Bool("register-only", false, "only add Register (ignore crud_register.go)")
	genWireCmd.Flags().Bool("crud-only", false, "only add RegisterCRUD (ignore register.go)")
	genCmd.AddCommand(genModuleCmd, genCrudCmd, genWireCmd)
	rootCmd.AddCommand(genCmd)
}

func runGenModule(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")
	skipWire, _ := cmd.Flags().GetBool("skip-wire")
	paths, err := gen.Module(moduleRoot, out, args[0], dry)
	if err != nil {
		return err
	}
	if dry {
		fmt.Println("dry-run — would write:")
	} else {
		fmt.Println("written:")
	}
	fmt.Println(strings.Join(paths, "\n"))
	if dry || skipWire {
		if !dry && skipWire {
			fmt.Println("\nWire skipped (--skip-wire). Run with wire enabled or edit the zatrano:wire markers manually.")
		}
		return nil
	}
	if wf, err := patchWire(moduleRoot, out, args[0], true, false); err != nil {
		fmt.Fprintf(os.Stderr, "wire: %v\n", err)
	} else {
		fmt.Println("\nWired:", filepath.ToSlash(wf))
		fmtWireFmt(moduleRoot, wf)
	}
	return nil
}

func runGenCrud(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	dry, _ := cmd.Flags().GetBool("dry-run")
	skipWire, _ := cmd.Flags().GetBool("skip-wire")
	paths, err := gen.CRUD(moduleRoot, out, args[0], dry)
	if err != nil {
		return err
	}
	if dry {
		fmt.Println("dry-run — would write:")
	} else {
		fmt.Println("written:")
	}
	fmt.Println(strings.Join(paths, "\n"))
	if dry || skipWire {
		if !dry && skipWire {
			fmt.Println("\nWire skipped (--skip-wire).")
		}
		return nil
	}
	if wf, err := patchWire(moduleRoot, out, args[0], false, true); err != nil {
		fmt.Fprintf(os.Stderr, "wire: %v\n", err)
	} else {
		fmt.Println("\nWired:", filepath.ToSlash(wf))
		fmtWireFmt(moduleRoot, wf)
	}
	return nil
}

func runGenWire(cmd *cobra.Command, args []string) error {
	out, _ := cmd.Flags().GetString("out")
	moduleRoot, _ := cmd.Flags().GetString("module-root")
	regOnly, _ := cmd.Flags().GetBool("register-only")
	crudOnly, _ := cmd.Flags().GetBool("crud-only")
	if regOnly && crudOnly {
		return fmt.Errorf("use only one of --register-only or --crud-only")
	}
	addReg, addCrud, modDir, err := gen.WireTargetsFromModuleDir(moduleRoot, out, args[0])
	if err != nil {
		return err
	}
	if regOnly {
		addCrud = false
	}
	if crudOnly {
		addReg = false
	}
	if !addReg && !addCrud {
		return fmt.Errorf("nothing to wire under %s", filepath.ToSlash(modDir))
	}
	wf, err := patchWire(moduleRoot, out, args[0], addReg, addCrud)
	if err != nil {
		return err
	}
	fmt.Println("Wired:", filepath.ToSlash(wf))
	fmtWireFmt(moduleRoot, wf)
	return nil
}

func fmtWireFmt(moduleRoot, wf string) {
	if err := goFmtWireFile(moduleRoot, wf); err != nil {
		fmt.Fprintf(os.Stderr, "go fmt (wire file): %v\n", err)
		return
	}
	rel, err := filepath.Rel(filepath.Clean(moduleRoot), filepath.Clean(wf))
	if err != nil {
		return
	}
	fmt.Println("Formatted:", filepath.ToSlash(rel))
}

func patchWire(moduleRoot, out, rawName string, addRegister, addCRUD bool) (string, error) {
	wf, err := gen.ResolveWireFile(moduleRoot)
	if err != nil {
		return "", err
	}
	modImport, err := gen.ModuleImportPath(moduleRoot)
	if err != nil {
		return "", err
	}
	pkg := gen.PackageName(rawName)
	rel := filepath.ToSlash(filepath.Join(out, pkg))
	if err := gen.WirePatch(wf, modImport, rel, pkg, addRegister, addCRUD); err != nil {
		return "", err
	}
	return wf, nil
}
