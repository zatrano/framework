# Generated modules

Place **`zatrano gen module <name>`** output here (`modules/<name>/`).

Each module exposes **`Register(*core.App, *fiber.App)`** (and optionally **`RegisterCRUD`** after `zatrano gen crud`).

By default the CLI **patches the wire file** and runs **`go fmt`** on it. Use **`--skip-wire`** if you only want files under `modules/` — then run **`zatrano gen wire <name>`** to patch later.

This directory is optional until you run the generator.
