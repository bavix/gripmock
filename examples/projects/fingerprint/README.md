# 👆 Fingerprint Service 👆  
**A priority-based fingerprint validation service built with protocol buffers and tested with GripMock**

## 📌 Overview  
This Fingerprint Service example demonstrates priority-based routing - a common pattern in real applications. Think of it like a VIP system where some users get special treatment. We use **protocol buffers** for the service definition and **GripMock** to test priority systems and validation scenarios that can be tricky to get right.  

## 🚀 Features  
✅ **Priority System** – High and low priority stub matching  
✅ **Fingerprint Validation** – Simple ID-based validation logic  
✅ **Specific vs General Matching** – Precise ID matching with fallback  
✅ **Priority-Based Routing** – Route specific users to high-priority handlers  
✅ **Stub-Driven Testing** – Validate priority behavior with YAML/JSON mocks  
✅ **Fallback Logic** – General fallback with specific high-priority cases  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Priority Matching**  
- 🛠️ **High Priority**: Tests specific fingerprint validation with priority 100  
- 🔄 **Low Priority**: Tests general fallback with priority 1  
- 🎯 **Priority Logic**: Validates stub selection based on priority values  
- 🔍 **Input Matching**: Tests different input matching strategies  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **Validation Logic**  
- ✅ **Positive Scenarios**: Successful fingerprint validation with priority  
- ❌ **Negative Scenarios**: Invalid fingerprints, priority conflicts  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service.proto`: **Protocol buffer service definition**  

```
examples/projects/fingerprint  
├── case_high_priority.gctf       # High priority test case
├── service.proto                 # Service definition
└── stubs.yaml                    # Mock responses with priority
```  

## 🛠️ Getting Started  
### Run the Application  
```bash
gripmock --stub examples/projects/fingerprint examples/projects/fingerprint/service.proto
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify)**:
```bash
grpctestify examples/projects/fingerprint/
```  

## 👆 Priority Matching Patterns  
This example shows you how to implement priority-based routing in practice:  
- **VIP User Handling**: Specific user IDs get high-priority treatment - like premium customers in a support system  
- **Fallback Strategy**: General users get default low-priority responses - everyone else gets standard service  
- **Priority Hierarchy**: Clear priority levels (100 vs 1) for decision making - make the logic obvious and maintainable  
- **Input Matching Strategy**: Combination of exact matching and general fallback - handle specific cases first, then general ones  
- **User Experience**: Different validation results based on user priority - VIPs get better service, others get standard responses  

## ⚠️ Important Notes  
- All methods are **unary** (no streaming support).  
- Demonstrates **priority system** and **stub matching logic**.  
- Tests **fallback behavior** with different priority levels.  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover **priority matching scenarios**  
- Fallback logic is properly tested  
- Priority system behavior is validated  

---

Made with ❤️ and protocol buffers 