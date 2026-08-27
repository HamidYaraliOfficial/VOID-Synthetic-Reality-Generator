// Command void-cli is VOID's terminal control surface: load a Versioned
// scenario/schema config file (YAML or JSON), run it against a fresh seeded
// Universe, and export the resulting synthetic dataset — all without the
// API server or UI running, for scripting and CI use.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"void-platform/backend/internal/config"
	"void-platform/backend/internal/entity"
	"void-platform/backend/internal/export"
	"void-platform/backend/internal/scenario"
	"void-platform/backend/internal/scheduler"
	"void-platform/backend/internal/simulation"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = cmdRun(os.Args[2:])
	case "generate":
		err = cmdGenerate(os.Args[2:])
	case "scheduler":
		err = cmdScheduler(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("void-cli " + version)
		return
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`VOID CLI — Synthetic Reality Generator (terminal control surface)

Usage:
  void-cli run --config scenario.yaml [--seed N] [--out-dir ./out] [--format json|jsonl|csv|yaml|xml|sql]
  void-cli generate --schema schema.json --count 1000 --out entities.json [--seed N] [--format json|jsonl|csv|yaml|xml|sql]
  void-cli scheduler status --hours hours.json
  void-cli version`)
}

// --- run ---------------------------------------------------------------

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "", "path to a scenario config file (.json/.yaml/.yml)")
	seed := fs.Int64("seed", 0, "override the scenario seed (0 = use config's own seed)")
	outDir := fs.String("out-dir", "out", "directory to write exported datasets into")
	format := fs.String("format", "json", "export format: json|jsonl|csv|yaml|xml|sql")
	waitSeconds := fs.Int("wait", 5, "seconds to let the scenario run before exporting results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}
	root, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	var sc scenario.Scenario
	if err := config.DecodeSpec(root.Spec, &sc); err != nil {
		return fmt.Errorf("decoding scenario spec: %w", err)
	}
	if sc.Name == "" {
		sc.Name = "cli-run"
	}
	if *seed != 0 {
		sc.Seed = *seed
	}

	var schemas []*entity.Schema
	if raw, ok := root.Metadata["schemas"]; ok {
		schemasJSON, _ := json.Marshal(raw)
		if err := json.Unmarshal(schemasJSON, &schemas); err != nil {
			return fmt.Errorf("decoding metadata.schemas: %w", err)
		}
	}

	u := simulation.NewUniverse("cli-universe", sc.Name, sc.Seed)
	for _, s := range schemas {
		if err := u.AddSchema(s); err != nil {
			return fmt.Errorf("schema %s: %w", s.Name, err)
		}
	}
	eng := simulation.NewEngine(u)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*waitSeconds+5)*time.Second)
	defer cancel()

	fmt.Printf("Running scenario %q (seed=%d, %d schemas, %d timeline actions)...\n", sc.Name, u.Seed, len(schemas), len(sc.Timeline))
	if err := eng.RunScenario(ctx, &sc); err != nil {
		return err
	}
	time.Sleep(time.Duration(*waitSeconds) * time.Second)
	eng.Stop()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	counts := u.EntityCounts()
	for typeName, count := range counts {
		entities := u.Collection(typeName).All()
		records := make([]export.Record, 0, len(entities))
		for _, e := range entities {
			rec := export.Record{"id": e.ID, "type": e.Type, "state": e.State}
			for k, v := range e.Attributes {
				rec[k] = v
			}
			records = append(records, rec)
		}
		path := fmt.Sprintf("%s/%s.%s", *outDir, typeName, *format)
		if err := export.ToFile(path, export.Format(*format), typeName, records); err != nil {
			return fmt.Errorf("exporting %s: %w", typeName, err)
		}
		fmt.Printf("  %-20s %8d entities -> %s\n", typeName, count, path)
	}
	snap, err := eng.SaveSnapshot(sc.Name)
	if err == nil {
		fmt.Printf("Snapshot saved: %s\n", snap.Path)
	}
	fmt.Println("Done.")
	return nil
}

// --- generate ------------------------------------------------------------

func cmdGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	schemaPath := fs.String("schema", "", "path to an entity schema JSON file")
	count := fs.Int("count", 1000, "number of entities to generate")
	out := fs.String("out", "entities.json", "output file path")
	seed := fs.Int64("seed", 42, "random seed for reproducibility")
	format := fs.String("format", "json", "export format: json|jsonl|csv|yaml|xml|sql")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *schemaPath == "" {
		return fmt.Errorf("--schema is required")
	}
	data, err := os.ReadFile(*schemaPath)
	if err != nil {
		return err
	}
	var sch entity.Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	u := simulation.NewUniverse("cli-generate", sch.Name, *seed)
	if err := u.AddSchema(&sch); err != nil {
		return err
	}
	entities, err := u.SpawnEntities(sch.Name, *count, func(done, total int) {
		fmt.Printf("\r  generating... %d/%d", done, total)
	})
	fmt.Println()
	if err != nil {
		return err
	}
	records := make([]export.Record, 0, len(entities))
	for _, e := range entities {
		rec := export.Record{"id": e.ID}
		for k, v := range e.Attributes {
			rec[k] = v
		}
		records = append(records, rec)
	}
	if err := export.ToFile(*out, export.Format(*format), sch.Name, records); err != nil {
		return err
	}
	fmt.Printf("Generated %d %s entities -> %s\n", len(entities), sch.Name, *out)
	return nil
}

// --- scheduler -------------------------------------------------------------

func cmdScheduler(args []string) error {
	if len(args) == 0 || args[0] != "status" {
		return fmt.Errorf("usage: void-cli scheduler status --hours hours.json")
	}
	fs := flag.NewFlagSet("scheduler status", flag.ExitOnError)
	hoursPath := fs.String("hours", "", "path to a business-hours JSON file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *hoursPath == "" {
		return fmt.Errorf("--hours is required")
	}
	data, err := os.ReadFile(*hoursPath)
	if err != nil {
		return err
	}
	var hours scheduler.BusinessHours
	if err := json.Unmarshal(data, &hours); err != nil {
		return err
	}
	status, err := hours.StatusAt(time.Now())
	if err != nil {
		return err
	}
	if status.IsOpen {
		fmt.Printf("OPEN now — closes in %s (at %s)\n", status.TimeUntilNextH, status.NextChangeAt.Format(time.RFC1123))
	} else {
		fmt.Printf("CLOSED now — opens in %s (at %s)\n", status.TimeUntilNextH, status.NextChangeAt.Format(time.RFC1123))
	}
	return nil
}
