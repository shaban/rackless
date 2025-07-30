# MC-SoFX Architecture Documentation

This directory contains architectural diagrams, flow charts, and design decisions for the MC-SoFX AudioUnit controller.

## Documentation Index

### System Design
- **[System Architecture](./system-architecture.md)** - High-level component overview and data flow
- **[Device Enumeration Flow](./device-enumeration-flow.md)** - Device discovery and capability detection process

### Component Specifications
- **AudioUnit Host Architecture** *(Coming Next)* - Plugin loading and audio processing flow
- **Frontend State Management** *(Planned)* - UI component interaction and state synchronization
- **Error Handling Strategy** *(Planned)* - Error recovery and user notification patterns

### Decision Records
- **[ADR-001: CGO Architecture](./adr-001-cgo-architecture.md)** *(Planned)* - Consolidated vs. modular native code
- **[ADR-002: Device Capability Strategy](./adr-002-device-capabilities.md)** *(Planned)* - Upfront vs. lazy capability detection
- **[ADR-003: JSON Interface Design](./adr-003-json-interface.md)** *(Planned)* - CGO data marshaling approach

## Diagram Tools

All diagrams use **Mermaid** syntax for version control and GitHub integration:
- Flowcharts for process flows
- State diagrams for component lifecycle  
- Sequence diagrams for interaction patterns
- Class diagrams for data structures

## Usage Guidelines

### For Development
1. **Before implementing new features** - Check existing flows and state machines
2. **When adding components** - Update system architecture diagram
3. **For debugging** - Reference error handling flows and state transitions

### For Testing
1. **State coverage** - Ensure all diagram states have corresponding tests
2. **Error paths** - Test all error transitions shown in diagrams  
3. **Integration points** - Validate component interfaces match specifications

### For Documentation
1. **Keep diagrams current** - Update when implementation changes
2. **Link to code** - Reference specific files and functions where applicable
3. **Version decisions** - Document architectural choices with rationale

## Current Status

| Component | Architecture Documented | Implementation Status |
|-----------|------------------------|----------------------|
| Device Enumeration | ✅ Complete | ✅ Implemented |
| System Overview | ✅ Complete | ✅ Foundation Ready |
| AudioUnit Hosting | ⚠️ In Progress | ❌ Not Started |
| Frontend Integration | ❌ Planned | ⚠️ Basic UI Only |
| Error Handling | ❌ Planned | ⚠️ Partial |

## Next Steps

1. **AudioUnit Host Flow** - Map plugin loading, configuration, and processing lifecycle
2. **Device Compatibility Matrix** - Define valid device combination rules
3. **Real-time Update Architecture** - Design device change notification system
4. **Testing Strategy Documentation** - Map test cases to architectural components

---
*Architecture Team: GitHub Copilot + User*  
*Last Updated: July 29, 2025*
