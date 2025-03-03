# Terminal RSS Reader

A feature-rich terminal-based RSS reader written in Go, with intelligent article recommendations based on your interests.

## Features

- 🎨 Beautiful terminal UI with vibrant colors and modern design
- 🔍 Full-text search across all articles
- 🎯 Smart article recommendations based on your interests
- ⌨️ Intuitive arrow key navigation
- 📱 Responsive terminal interface
- 🔄 Automatic feed updates
- 🏷️ Interest-based scoring system
- 📊 Sort articles by relevance or date
- 🌐 Open articles directly in your browser
- 💾 Automatic state persistence
- 🎯 Weighted interest system
- 📈 Interest decay over time
- 🔒 Secure configuration storage

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

- ↑/↓ arrows to navigate items
- ←/→ arrows to change pages
- Enter to select/view
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

- Navigate through articles using arrow keys
- Press Enter to view full article
- `o` to open in browser
- `y` to mark as interesting (improves recommendations)
- `n` to skip to next article

### Search and Recommendations

- `s` to search articles
- `r` to view recommended articles
- Sort by relevance or date using `s`
- Quick navigation with `o[number]` to open specific articles

### Managing Interests

- `i` to manage interests
- Add new interests with custom weights (0.1-10.0)
- Higher weights give stronger recommendations
- Interests are automatically updated as you read
- Automatic interest decay over time

## Configuration

The application stores its configuration in `~/.rss-reader/`:
- `profile.json`: User preferences and interests
- `feeds.txt`: RSS feed URLs

### Default Feeds

The application comes with some default RSS feeds:
- Less News (https://lessnews.dev/rss.xml)
- Go Blog (https://blog.golang.org/feed.atom)
- Hacker News (https://news.ycombinator.com/rss)
- Dev.to (https://dev.to/feed)

## Project Structure

```
.
├── internal/
│   ├── config/     # Configuration management
│   ├── models/     # Data models and types
│   ├── storage/    # Data persistence
│   └── ui/         # Terminal UI styles
├── main.go         # Application entry point
├── go.mod          # Go module file
└── README.md       # Documentation
```

## Dependencies

- [gofeed](https://github.com/mmcdole/gofeed): RSS feed parsing
- [lipgloss](https://github.com/charmbracelet/lipgloss): Terminal styling
- [term](golang.org/x/term): Terminal input handling

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with modern Go practices and patterns
- Terminal UI inspired by modern CLI applications
- Uses semantic versioning for releases

## Author

Jason Dittmer ([@thedittmer](https://github.com/thedittmer)) 