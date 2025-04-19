# 🔄 UnitConvertor 🔄  
**A robust unit conversion service built with protocol buffers and tested with GripMock**

## 📌 Overview  
UnitConvertor is a lightweight application designed to seamlessly convert between different units of measurement. It leverages **protocol buffers** for service definition and employs **GripMock** for rigorous testing of both API contracts and edge cases.  

## 🚀 Features  
✅ **Unary Methods Only** – Simple, fast, and predictable conversions  
✅ **No Well-Known Types** – Clean, dependency-free proto definitions  
✅ **Stub-Driven Testing** – Validate behavior with YAML/JSON mocks  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Proto Descriptor Integrity**  
- 🛠️ **Service Creation**: Validates that the service is built **only from `.pb` descriptors** (not `.proto` sources).  
- 🗑️ **Pre-Test Cleanup**: Deletes all `*.proto.src` files to prevent accidental compilation from source code.  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **Behavior Validation**  
- ✅ **Positive Scenarios**: Happy-path conversions (e.g., meters → kilometers)  
- ❌ **Negative Scenarios**: Invalid units, out-of-range values, malformed requests  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service.proto.src`: **Source proto file** (deleted before testing)  
- `service.pb`: **Compiled proto descriptor**  

```
examples/projects/unitconverter  
├── convert_length  
│   ├── case_missing_from_unit.gctf  
│   ├── case_missing_to_unit.gctf  
│   ├── case_success.gctf  
│   ├── stub_multi.yml  
│   └── stub_single.yml  
├── convert_temperature  
│   ├── case_invalid_conversion.gctf  
│   ├── case_missing_from_unit.gctf  
│   ├── case_success.gctf  
│   ├── stub_multi.yaml  
│   └── stub_single.yaml  
├── convert_volume  
│   ├── case_devision_by_zero.gctf  
│   ├── case_missing_from_unit.gctf  
│   ├── case_success.gctf  
│   └── stub.yml  
├── convert_weight  
│   ├── case_missing_to_unit.gctf  
│   ├── case_negative_value.gctf  
│   ├── case_success.gctf  
│   ├── stub_multi.json  
│   └── stub_single.json  
├── service.pb  
└── service.proto.src  
```  

## 🛠️ Getting Started  
### Run the Application  
#### Option 1: Direct Execution  
```bash
go run main.go --stub examples/projects/unitconverter examples/projects/unitconverter/service.pb
```

#### Option 2: Pre-Built Binary (using GripMock)  
```bash
gripmock --stub examples/projects/unitconverter examples/projects/unitconverter/service.pb
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify)**:  
```bash
grpctestify examples/projects/unitconverter/
```  

## ⚠️ Important Notes  
- The pipeline **deletes `service.proto.src`** before testing to enforce descriptor-based builds.  
- All methods are **unary** (no streaming support).  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover both **positive** and **negative** cases  
- No well-known types are introduced in proto files  

---

Made with ❤️ and protocol buffers  
