# ✅ Validator Service ✅  
**A comprehensive data validation service with multiple validation types built with protocol buffers and tested with GripMock**

## 📌 Overview  
This Validator Service example shows you how to build comprehensive data validation systems. It's like a bouncer at a club that checks IDs, but for data - it validates emails, phone numbers, URLs, and numeric ranges. We use **protocol buffers** for the service definition and **GripMock** to test validation logic and edge cases that can be surprisingly tricky.  

## 🚀 Features  
✅ **Multiple Validation Types** – Email, URL, phone number, and numeric range validation  
✅ **OneOf Support** – Flexible input handling with oneof fields  
✅ **Range Validation** – Min/max value validation for numeric types  
✅ **Regex Validation** – Pattern-based validation with regular expressions  
✅ **Error Messages** – Descriptive error messages for validation failures  
✅ **Stub-Driven Testing** – Validate validation behavior with YAML/JSON mocks  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Validation Scenarios**  
- 🛠️ **Valid Range**: Tests successful validation within specified ranges  
- 🚫 **Invalid Range**: Tests validation failures outside specified ranges  
- 🔍 **Regex Validation**: Tests pattern-based validation (email, URL, phone)  
- 📊 **Numeric Validation**: Tests min/max value constraints  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **Validation Logic**  
- ✅ **Positive Scenarios**: Successful validation across all types  
- ❌ **Negative Scenarios**: Validation failures, edge cases, malformed data  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service.proto`: **Protocol buffer service definition**  

```
examples/projects/validator  
├── case_invalid_range.gctf       # Invalid range test
├── case_regexp_range.gctf        # Regex validation test
├── case_valid_range.gctf         # Valid range test
├── service.proto                 # Service definition
└── stubs.yaml                    # Mock responses
```  

## 🛠️ Getting Started  
### Run the Application  
```bash
gripmock --stub examples/projects/validator examples/projects/validator/service.proto
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify)**:  
```bash
grpctestify examples/projects/validator/
```  

## ✅ Validation Patterns  
This example shows you how to build robust validation systems in practice:  
- **Multi-Type Validation**: Email, URL, phone number, and numeric range validation - because real applications need to validate all kinds of data  
- **OneOf Field Handling**: Flexible input types using protobuf oneof fields - handle different data types in a single service  
- **Range Validation**: Min/max boundary checking for numeric values - like ensuring age is between 0 and 150  
- **Regex Pattern Matching**: Complex pattern validation with regular expressions - for things like phone number formats  
- **Error Handling**: Descriptive error messages for different validation failures - users need to know what went wrong  
- **Validation Strategies**: Combination of exact matching and pattern matching - sometimes you need both  
- **Edge Case Testing**: Boundary conditions and invalid input scenarios - because edge cases are where bugs hide  

## ⚠️ Important Notes  
- All methods are **unary** (no streaming support).  
- Demonstrates **oneof field handling** and **validation logic**.  
- Tests **multiple validation types** and **edge cases**.  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover **all validation types**  
- Edge cases are properly tested  
- Validation logic is comprehensive  

---

Made with ❤️ and protocol buffers 