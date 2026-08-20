# Bumblebee Maps

Reusable app for generating a "bumblebee map" from Strava / bike data.

The backend is Go/Chi, and the frontend is embedded bun/vue.

## Development Setup

This project uses [mise](https://mise.jdx.co/) to manage development tools and run tasks.

### Prerequisites

Ensure you have `mise` installed. Then, trust the configuration, install the required tools (Go, Bun, Air, DuckDB), and initialize the DuckDB spatial extension:

```bash
mise trust
mise install
mise run setup
```

### Key Commands

Run tasks using `mise run <task>` (or the shorthand `mise r <task>`):

* **Start Development Servers (Backend + Frontend concurrently):**
  ```bash
  mise run dev
  ```
  This runs the backend live-reload server (`air`) and the Vite frontend dev server concurrently.

* **Build for Production:**
  ```bash
  mise run build
  ```
  This builds the Vite/Vue frontend and compiles the Go backend, embedding the frontend assets into the final binary (`bin/map-tools`).

* **Run Production Build:**
  ```bash
  mise run run
  ```
  Builds and runs the compiled binary.


### Persistent Upload Storage

Processed GeoParquet uploads are stored outside the repository so the compiled app can be launched from anywhere without changing where your saved datasets live.

- **macOS:** `~/Library/Application Support/MapTools/data`
- **Linux:** `$XDG_DATA_HOME/MapTools/data`, or `~/.local/share/MapTools/data` when `XDG_DATA_HOME` is unset
- **Windows:** `%AppData%\MapTools\data`

You can override the storage location with `MAPTOOLS_DATA_DIR=/absolute/path/to/data`.

Short-lived ZIP and GeoJSON scratch files are written to the operating system temp directory and deleted after processing. The Uploads page includes an **Open File** button beside each saved dataset so you can reveal it in Finder or your file manager. Older `.parquet` files that already exist in a repo-local `tmp/` directory still appear in the Uploads page so existing local runs stay usable.
* **Format Code:**
  ```bash
  mise run fmt
  ```
  Formats all Go and Vue/TS source code in the repository.

* **Clean Build Artifacts:**
  ```bash
  mise run clean
  ```
  Removes the compiled binary and built frontend files.

### Individual Dev Tasks

If you want to run the frontend or backend dev servers separately:
- **Frontend Dev Server:** `mise run dev-frontend` (runs on port 5173, proxies `/api` to port 8080)
- **Backend Dev Server:** `mise run dev-backend` (runs the Go backend using `air` on port 8080)
