# AI Coding Assistant Instructions for PDU Calibration Tool

## Project Overview
This is a Go-based desktop application for calibrating Power Distribution Units (PDUs) using a graphical interface. It communicates with a standard power source and PDU devices via serial ports using Modbus protocol, performs automated calibration tests at different load levels, and exports results to Excel.

## Architecture & Key Components

### Core Structure
- **main.go**: Entry point, initializes GUI and runs calibration loop
- **check/check.go**: Core calibration workflow (Run() function)
- **gui/**: Fyne-based UI with navigation, tables, and info panels
- **devices/power_source/** & **devices/pdu/**: Device communication drivers
- **communications/modbus/**: Modbus protocol implementation
- **config/**: YAML configuration loading
- **types/types.go**: Shared data structures (CalibrationData, CheckResult)

### Data Flow
1. User selects serial ports via GUI dropdowns
2. Scans product SN via input dialog
3. Calibration loop: configure power source → calibrate PDU → test at 6A/3A loads → check thresholds → export Excel → shutdown

### Communication Patterns
- Serial ports: Standard source (default 38400 baud) and PDU (configurable)
- Modbus RTU for device control and data reading
- Commands defined in `configs/serial.yaml` as hex byte arrays
- Data scaling: Voltage÷10, Current÷10, Power÷10, PF÷1000 for display/calculation

## Critical Developer Workflows

### Building & Running
- **Development**: `go run .` (runs directly from source)
- **Build**: `build.bat` (generates rsrc.syso from icon, embeds version/date)
- **Clean**: `build.bat clean`
- **Package**: `build.bat package` (creates publish/ directory with exe + configs)

### Debugging
- Enable via `configs/debug.yaml`: Modbus=1 for protocol logs, BreakPoint=1 for pauses
- Terminal output shows calibration progress and data values
- GUI updates in real-time during tests

### Configuration
- **thresholds.yaml**: Voltage/Current/Power/PF tolerance limits
- **serial.yaml**: Port names, baud rates, command hex arrays, timeouts
- **debug.yaml**: Debug switches

## Project-Specific Patterns & Conventions

### Data Handling
- **Scaling**: Raw device values need division (V/10, I/10, P/10, PF/1000) for real units
- **Reactive Power**: Computed from PF using `ComputeReactivePowerFromPF(voltage, current, pf)`
- **Energy Calculation**: `ComputeEnergyWh(power, duration)` and `ComputeReactiveEnergyVARh(reactivePower, duration)`
- **Threshold Checking**: Absolute differences compared to config.Thresholds

### Error Handling
- Device connections: Open/Close pattern with defer cleanup
- Calibration failures: Return errors, show dialogs, log to terminal
- Serial timeouts: Configurable via ReadTimeoutMs

### GUI Updates
- Real-time status: `window.SetMultiFuncButtonText("status...")`
- Table refresh: `table.UpdateCalibrationTable(calibrationData)`
- Dialogs: `customWidget.ShowDialog()` for results/errors
- Input prompts: `customWidget.ShowInputDialog()` for SN scanning

### Excel Export
- Uses `util.ExportToExcel(sn, test1, test2, path)` 
- Includes all CalibrationData fields plus computed energies
- Path defaults to current directory if empty

### Testing Sequence
1. **6A Test**: Set source to 220V/6A → Calibrate PDU → Read/compare → Check thresholds
2. **3A Test**: Set source to 200V/3A → Read/compare → Check thresholds  
3. **Cleanup**: Clear PDU energy → Export Excel → Stop power source

### Device Commands
- Power source: "single_220_6", "single_200_3", "start_L1", "three_stop"
- PDU: Calibration commands (C2/D1), data read, energy clear
- Commands mapped in serial.yaml as named hex arrays

## Key Files for Understanding Patterns
- `check/check.go`: Complete calibration flow with timing/delays
- `types/types.go`: All data structures and result enums  
- `config/config.go`: Configuration loading and validation
- `devices/power_source/power_source.go`: Serial communication and command sending
- `gui/window/navigate.go`: UI navigation structure
- `util/excel.go`: Export format and calculations

## Common Pitfalls
- Serial port paths: Use `\\.\COMx` format on Windows
- Data scaling: Always apply divisions for display/storage
- Timing: 3-8 second delays needed for power stabilization
- Command verification: Always read back after sending commands
- GUI thread safety: Updates must happen on main thread

## Adding New Features
- New devices: Add to `devices/` with Open/Close/Read/Calibrate functions
- New tests: Extend `check.Run()` with additional load configurations  
- New UI: Add to `gui/window/` and wire into navigation
- New commands: Add hex arrays to `serial.yaml` and implement in device drivers