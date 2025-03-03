# Terminal RSS Reader

A feature-rich terminal-based RSS reader written in Go, with intelligent article recommendations based on your interests.

![Terminal RSS Reader](screenshot.png) <!-- You might want to add a screenshot here -->

## Features

- 🎨 Beautiful terminal UI with vibrant colors and modern design
- 🔍 Full-text search across all articles
- 🎯 Smart article recommendations based on your interests
- ⌨️ Vim-style keyboard navigation (h/j/k/l)
- 📱 Responsive terminal interface
- 🔄 Automatic feed updates
- 🏷️ Interest-based scoring system
- 📊 Sort articles by relevance or date
- 🌐 Open articles directly in your browser

## Installation

```bash
# Clone the repository
git clone https://github.com/thedittmer/rss-reader.git

# Navigate to the project directory
cd rss-reader

# Install dependencies
go mod download

# Build the application
go build

# Run the application
./rss-reader
```

## Usage

### Basic Navigation

- Arrow keys or vim-style keys (h/j/k/l) to navigate
- Enter to select
- b to go back
- q to quit
- h for help in any screen

### Managing Feeds

- `f` in main menu to manage feeds
- Add new feeds with `a`
- Remove feeds with `r`
- Feeds are automatically updated on startup
- Manual refresh with `x` in main menu

### Reading Articles

- Navigate through articles using arrow keys or j/k
- Press Enter to view full article
- `o` to open in browser
- `y` to mark as interesting (improves recommendations)
- `n` to skip to next article

### Search and Recommendations

- `s` to search articles
- `r` to view recommended articles
- Sort by relevance or date
- Quick navigation with `o[number]` to open specific articles

### Managing Interests

- `i` to manage interests
- Add new interests with custom weights
- Higher weights give stronger recommendations
- Interests are automatically updated as you read

## Configuration

The application stores its configuration in `~/.rss-reader/`:
- `profile.json`: User preferences and interests
- `feeds.txt`: RSS feed URLs

## Dependencies

- [gofeed](https://github.com/mmcdole/gofeed): RSS feed parsing
- [lipgloss](https://github.com/charmbracelet/lipgloss): Terminal styling
- [term](golang.org/x/term): Terminal input handling

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the design aesthetics of pnpm
- Built with modern Go practices and patterns
- Terminal UI inspired by modern CLI applications

## Author

Jason Dittmer ([@thedittmer](https://github.com/thedittmer)) 