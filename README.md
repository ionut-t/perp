# perp

**perp** is a TUI application for interacting with PostgreSQL databases.

## Features

- **Cross-platform**: works on Linux, macOS and Windows.
- **Multiple database servers**: connect to multiple database servers.
- **Run queries**: run queries and view results.
- **LLM integration**:
  - Use `/ask` to translate natural language to SQL.
  - Use `/explain` to explain a SQL query.
  - Use `/optimise` to optimise a SQL query.
  - Use `/fix` to fix a SQL query.
  - Use `/add` to add tables to the LLM context.
  - Use `/remove` to remove tables from the LLM context.
  - Enable/disable database schema in LLM queries.
  - Set the LLM model to use for queries.
  - View LLM logs.
- **PSQL commands**: run psql commands (e.g. `\d`, `\dt`, `\l`, etc.).
- **Export data**:
  - Export all data returned by the query as JSON/CSV to a file.
  - Export specific rows to a file.
  - Manage exported data in the export view (accessible with `g`):
    - View a list of exported files.
    - View and edit exported files.
    - Rename and delete exported files.
- **Clipboard**:
  - Yank/copy selected cell to clipboard.
  - Yank/copy selected row as JSON to clipboard.
- **Editor**:
  - Vim keybindings.
  - Visual mode for selecting text.
  - Paste from clipboard.
  - Undo/redo.
- **History**:
  - View and navigate query history.
- **Database schema**:
  - View database schema.
  - View LLM shared schema.
- **Command palette**: access commands by pressing `:`.
- **Server management**:
  - Create, edit, and delete server connections.
  - View server details.

## Instalation

```sh
go install github.com/ionut-t/perp
```

Or install the binary from the [Releases](https://github.com/ionut-t/perp/releases) page.

## Key Bindings

| Key                     | Action                         |
| ----------------------- | ------------------------------ |
| `i`                     | Enter insert mode              |
| `esc`                   | Return to normal mode          |
| `alt+enter/ctrl+s`      | Send query                     |
| `y`                     | Yank/copy selected cell        |
| `Y`                     | Yank/copy selected row as JSON |
| `p`                     | Paste in the editor            |
| `export 1,2,3 data.csv` | Export selected rows as CSV    |
| `export * data.json`    | Export all rows as JSON        |

A complete list of key bindings and commands is accessible through the help menu.

## Usage

1. Start the application:
   ```sh
   perp
   ```
2. Create/Select a server and connect.
3. Write SQL queries or use `/ask` for LLM assistance.
4. Navigate results and use key bindings for actions.

## Configuration

The configuration file is located at `~/.perp/.config.toml`.

The following keys are available:

| Key                       | Description                                                               |
| ------------------------- | ------------------------------------------------------------------------- |
| `EDITOR`                  | The editor to use for editing config, LLM instructions and exported data. |
| `MAX_HISTORY_LENGTH`      | The maximum number of history entries to keep.                            |
| `MAX_HISTORY_AGE_IN_DAYS` | The maximum number of days to keep history entries.                       |
| `LLM_PROVIDER`            | The LLM provider to use. It can be set to `Gemini` or `VertexAI`.         |
| `LLM_MODEL`               | The LLM model is required for both `Gemini` and `VertexAI`.               |

The `config` command can be used to manage the configuration:

```sh
perp config
```

This will open the configuration file in the default editor.

Some configuration keys can be set using command-line flags:

```sh
perp config -e vim
perp config -p Gemini -m gemini-2.5-pro
```

### Environment Variables

For LLM integration, you need to set the appropriate environment variables based on the chosen LLM provider:

- **Gemini**: Set the `GEMINI_API_KEY` environment variable to your Gemini API key.
- **VertexAI**: Set the `VERTEXAI_PROJECT_ID` and `VERTEXAI_LOCATION` environment variables.

## LLM Instructions

Custom instructions for the LLM can be added by running the following command:

```sh
perp llm-instructions
```

This will open the LLM instructions file in the default editor. The file is located at `~/.perp/llm_instructions.md`.

The default instructions can be found [here](internal/config/llm_instructions.md).

## Development

- Written in Go
- Uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI

## Acknowledgement

| Library                                                  | Purpose                                         |
| -------------------------------------------------------- | ----------------------------------------------- |
| [pgx](https://github.com/jackc/pgx)                      | PostgreSQL Driver and Toolkit                   |
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Building the TUI                                |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss)   | Styling terminal UI components                  |
| [Bubbles](https://github.com/charmbracelet/bubbles)      | Reusable TUI components                         |
| [glamour](https://github.com/charmbracelet/glamour)      | Rendering markdown in the terminal              |
| [clipboard](https://github.com/atotto/clipboard)         | Clipboard operations                            |
| [editor](https://github.com/ionut-t/goeditor)            | The editor component used for running queries   |
| [table](https://github.com/ionut-t/gotable)              | The table component used for displaying results |

## License

[MIT](LICENSE)

