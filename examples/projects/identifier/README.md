# 🆔 Identifier Service 🆔  
**A UUID processing service with array handling built with protocol buffers and tested with GripMock**

## 📌 Overview  
This Identifier Service example shows you how to handle complex data processing with UUIDs in different formats. It's like a data transformation service that can work with UUIDs whether they come as strings, integers, or binary data. We use **protocol buffers** for the service definition and **GripMock** to test array processing and UUID validation scenarios that can be surprisingly complex.  

## 🚀 Features  
✅ **UUID Processing** – Handle multiple UUID formats (int64, bytes, string)  
✅ **Array Operations** – Process repeated UUID arrays efficiently  
✅ **Order Handling** – Strict and unstrict array order validation  
✅ **Template Functions** – UUID conversion helpers (uuid2int64, uuid2base64)  
✅ **Timestamp Processing** – Optional request/response timestamp handling  
✅ **Stub-Driven Testing** – Validate array behavior with YAML/JSON mocks  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Array Processing**  
- 🛠️ **Strict Ordered**: Tests UUID arrays with strict order requirements  
- 🔄 **Unstrict Ordered**: Tests UUID arrays with flexible order handling  
- 📊 **Multiple Formats**: Tests int64, bytes, and string UUID formats  
- ⏰ **Timestamp Handling**: Tests optional timestamp processing  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **UUID Validation**  
- ✅ **Positive Scenarios**: Successful UUID processing across formats  
- ❌ **Negative Scenarios**: Invalid UUIDs, array order violations  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service.proto`: **Protocol buffer service definition**  

```
examples/projects/identifier  
├── case_1_success_unstrict_ordered.gctf    # Unstrict order test 1
├── case_2_success_unstrict_ordered.gctf    # Unstrict order test 2
├── case_3_success_unstrict_ordered.gctf    # Unstrict order test 3
├── case_success_strict_ordered.gctf        # Strict order test
├── service.proto                           # Service definition
└── stubs.yaml                              # Mock responses
```  

## 🛠️ Getting Started  
### Run the Application  
```bash
gripmock --stub examples/projects/identifier examples/projects/identifier/service.proto
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify-rust)**:
```bash
grpctestify examples/projects/identifier/
```  

## 🆔 UUID Processing Patterns  
This example shows you how to handle UUIDs in real-world scenarios:  
- **Multi-Format Support**: int64, base64, and string UUID representations - because different systems use different formats  
- **Template Functions**: Dynamic UUID conversion using `{{ uuid2int64 }}` and `{{ uuid2base64 }}` - powerful templating for data transformation  
- **Array Order Flexibility**: Both strict and unstrict array ordering for UUID lists - sometimes order matters, sometimes it doesn't  
- **Timestamp Correlation**: Request/response timestamp tracking for processing - useful for debugging and monitoring  
- **Process Tracking**: Unique process IDs for each UUID batch processing - helps with tracing and debugging  
- **Zero UUID Handling**: Special handling for null/zero UUIDs (`00000000-0000-0000-0000-000000000000`) - because null values happen in real data  

## ⚠️ Important Notes  
- All methods are **unary** (no streaming support).  
- Demonstrates **array processing** and **UUID format handling**.  
- Tests **order sensitivity** in array operations.  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover **array processing scenarios**  
- UUID format handling is properly tested  
- Order validation logic is comprehensive  

---

Made with ❤️ and protocol buffers 