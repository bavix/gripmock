# 🔊 Echo Service 🔊  
**A versioned echo service demonstrating API evolution built with protocol buffers and tested with GripMock**

## 📌 Overview  
This Echo Service example demonstrates how to handle API versioning and evolution - something every real API eventually needs. It shows you how to maintain backward compatibility while adding new features. We use **protocol buffers** for the service definition and **GripMock** to test version compatibility and message handling across different API versions.  

## 🚀 Features  
✅ **API Versioning** – Multiple service versions (v1, v2)  
✅ **Message Echo** – Simple request-response echo functionality  
✅ **Case Sensitivity** – Tests different method naming conventions  
✅ **Package Namespacing** – Full package names in service definitions  
✅ **Method Overloading** – Same functionality with different method names  
✅ **Stub-Driven Testing** – Validate version behavior with YAML/JSON mocks  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Version Compatibility**  
- 🛠️ **V1 Service**: Tests original API version behavior  
- 🔄 **V2 Service**: Tests evolved API version behavior  
- 📝 **Method Variations**: Tests case-sensitive method names  
- 🔍 **Empty Messages**: Validates empty message handling  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **Message Validation**  
- ✅ **Positive Scenarios**: Successful message echoing across versions  
- ❌ **Negative Scenarios**: Version incompatibilities, malformed requests  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service_v*.proto`: **Protocol buffer service definitions**  

```
examples/projects/echo  
├── case_v1_empty.gctf            # V1 empty message test
├── case_v1_lower.gctf            # V1 lowercase method test
├── case_v1_upper.gctf            # V1 uppercase method test
├── case_v2_lower.gctf            # V2 lowercase method test
├── case_v2_upper.gctf            # V2 uppercase method test
├── service_v1.proto              # V1 service definition
├── service_v2.proto              # V2 service definition
├── stubs_v1.yml                  # V1 mock responses
└── stubs_v2.yml                  # V2 mock responses
```  

## 🛠️ Getting Started  
### Run the Application  
```bash
gripmock --stub examples/projects/echo examples/projects/echo/service_v1.proto
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify-rust)**:  
```bash
grpctestify examples/projects/echo/
```  

## 🔊 API Evolution Patterns  
This example shows you how to handle API evolution in the real world:  
- **Version Compatibility**: Testing both v1 and v2 service versions - because breaking changes happen  
- **Method Naming Conventions**: Case-sensitive method names (SendMessage vs sendMessage) - because naming matters in APIs  
- **Package Evolution**: Full package namespacing (`com.bavix.echo.v1`) - for proper version isolation  
- **Backward Compatibility**: Maintaining functionality across versions - so old clients don't break  
- **Fallback Behavior**: Generic responses for unmatched requests - graceful degradation is important  

## ⚠️ Important Notes  
- All methods are **unary** (no streaming support).  
- Demonstrates **API versioning** and **evolution patterns**.  
- Tests **case sensitivity** in method names.  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover **version compatibility**  
- API evolution patterns are properly tested  
- Backward compatibility is maintained  

---

Made with ❤️ and protocol buffers 