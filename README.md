# Markdown to PDF Converter

## Overview
This project provides a command-line tool and API server to convert Markdown files into PDF format, enabling users to create high-quality PDFs from their Markdown documentation easily.

## Features
- Convert Markdown (.md) files to PDF.
- Supports various Markdown syntax.
- Customizable PDF formatting options.
- HTTP API server for reporting services.

## Installation
To install the Markdown to PDF Converter, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/owner/reporting-v2.git
   cd reporting-v2
   ```
2. Install dependencies and build:
   ```bash
   go mod download
   go build -o reporting ./cmd/reporting
   ```

## Usage

### CLI Tool
To generate a PDF, run:
```bash
./reporting --title "My Document" --author "Your Name"
```

### API Server
To start the API server:
```bash
go run main.go
```

## Options
You can customize the PDF generation by providing optional parameters:
- `--title`: Title of the PDF document.
- `--author`: Author of the document.
- `--theme`: Theme of the document (`default` or `dark`).

## Contributing
We welcome contributions! Please submit a pull request or open an issue to discuss changes.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact
For any inquiries, please contact:
- **Denis Axon** - denis.axon@example.com
