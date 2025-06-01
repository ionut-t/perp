# perp

**perp** is a TUI application for interacting with PostgreSQL databases.

## Features

- Connect to multiple database servers
- Execute SQL queries and view results in a table
- Copy/yank cell values to clipboard
- Export query results as JSON
- View and navigate database schema
- Integrated LLM assistant via `/ask` command (requires Gemini API key to be provided in the config file)

## Instalation

```sh
go install github.com/ionut-t/perp
```

## Key Bindings

| Key         | Action                      |
| ----------- | --------------------------- |
| `i`         | Enter insert mode           |
| `esc`       | Return to normal mode       |
| `alt+enter` | Send query                  |
| `y`         | Yank/copy selected cell     |
| `p`         | Paste in the editor         |
| `e`         | Export selected row as JSON |
| `E`         | Export all results as JSON  |

## Usage

1. Start the application:
   ```sh
   perp
   ```
2. Create/Select a server and connect.
3. Write SQL queries or use `/ask` for LLM assistance.
4. Navigate results and use key bindings for actions.

## Development

- Written in Go
- Uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) for TUI

## Acknowledgement

| Library                                                  | Purpose                        |
| -------------------------------------------------------- | ------------------------------ |
| [pgx](https://github.com/jackc/pgx)                      | PostgreSQL Driver and Toolkit  |
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Building the TUI               |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss)   | Styling terminal UI components |
| [Bubbles](https://github.com/charmbracelet/bubbles)      | Reusable TUI components        |
| [clipboard](https://github.com/atotto/clipboard)         | Clipboard operations           |

## License

[MIT](LICENSE)

