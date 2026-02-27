# Markdown to PDF Converter

## Overview
This project provides a command-line tool to convert Markdown files into PDF format, enabling users to create high-quality PDFs from their Markdown documentation easily.

## Features
- Convert Markdown (.md) files to PDF.
- Supports various Markdown syntax.
- Customizable PDF formatting options.

## Installation
To install the Markdown to PDF Converter, follow these steps:

1. Clone the repository:
   ```bash
   git clone https://github.com/owner/reporting-v2.git
   cd reporting-v2
   ```
2. Install the required dependencies:
   ```bash
   npm install
   ```

## Usage
To convert a Markdown file to PDF, run the following command:
```bash
node convert.js <input_file.md> <output_file.pdf>
```

### Example
```bash
node convert.js README.md output.pdf
```

## Options
You can customize the PDF generation by providing optional parameters:
- `--format`: Specify the format of the output PDF.
- `--css`: Provide a CSS file for custom styling.

## Contributing
We welcome contributions! Please submit a pull request or open an issue to discuss changes.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact
For any inquiries, please contact:
- **Denis Axon** - denis.axon@example.com
