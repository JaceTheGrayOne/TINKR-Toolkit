# TINK.R Toolkit

A software multi-tool designed to streamline the use of various tools required for video game modding and data mining.

## Currently Supported Tools

- **Retoc** - Zen asset packer/unpacker


## Features

- **Automatic Mod Discovery** - Scans your mods directory and lists all available mods
- **Parallel Building** - Build multiple mods simultaneously
- **Quick Hotkeys** - Press 0-9 to instantly build specific mods
- **Multi-Select** - Choose multiple mods to build in batch
- **User Config** - First-run setup with path normalization and validation


### Quick Start

1. **Download the latest release** from [Releases](https://github.com/JaceTheGrayOne/TINKR-Toolkit/releases)
2. **Extract** `TINKR-Toolkit.exe` and the `retoc/` folder to your desired location
3. **Run** `TINKR-Toolkit.exe`
4. **Configure** your paths on first run

### Building from Source
```bash
git clone https://github.com/jacethegrayone/tinkr-toolkit.git
cd tinkr-toolkit
go build -o tinkr-toolkit.exe