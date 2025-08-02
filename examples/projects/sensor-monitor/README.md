# 📡 Sensor Monitor 📡  
**A sensor data monitoring service with streaming capabilities built with protocol buffers and tested with GripMock**

## 📌 Overview  
This Sensor Monitor example demonstrates how to build IoT and real-time monitoring systems. Think of it like a smart home system that continuously streams data from temperature sensors, security cameras, or any IoT device. We use **protocol buffers** for the service definition and **GripMock** to test streaming scenarios and sensor data processing that can be challenging to get right.  

## 🚀 Features  
✅ **Real-time Monitoring** – Continuous sensor data streaming  
✅ **Data Processing** – Handle various sensor data formats  
✅ **Streaming Support** – Server-side streaming for live data  
✅ **IoT Integration** – Internet of Things sensor data patterns  
✅ **Time-Series Data** – Continuous data flow with timestamps  
✅ **Stub-Driven Testing** – Validate streaming behavior with YAML/JSON mocks  

## 🔍 Test Cases (GripMock)  
The CI pipeline enforces strict testing standards:  

### 1. **Sensor Data Streaming**  
- 🛠️ **Data Collection**: Tests continuous sensor data collection  
- 📊 **Data Processing**: Validates sensor data format handling  
- 🔄 **Real-time Updates**: Tests live data streaming capabilities  
- 📈 **Performance Monitoring**: Tests streaming performance under load  

### 2. **Stub File Scenarios**  
| Type                | Description                                  | Supported Formats          |  
|----------------------|----------------------------------------------|----------------------------|  
| Single Stub          | Test with one mock response file             | `.yaml`, `.yml`, `.json`   |  
| Multiple Stubs       | Combine multiple stubs for complex flows    | `.yaml`, `.yml`, `.json`   |  
| Multistab Files      | Define multiple mock responses in one file  | `.yaml`, `.yml`, `.json`   |  

### 3. **Monitoring Validation**  
- ✅ **Positive Scenarios**: Successful sensor data streaming and processing  
- ❌ **Negative Scenarios**: Sensor failures, data corruption, connection issues  

## 📂 Project Structure  
**File descriptions**:  
- `*.json`/`*.yaml`/`*.yml`: **Stub files** for mock responses  
- `*.gctf`: **Test case definitions**  
- `service.proto`: **Protocol buffer service definition**  

```
examples/projects/sensor-monitor  
└── stubs/                        # Stub files directory
```  

## 🛠️ Getting Started  
### Run the Application  
```bash
gripmock --stub examples/projects/sensor-monitor examples/projects/sensor-monitor/service.proto
```

### Run Tests  
Execute tests using **[grpctestify](https://github.com/gripmock/grpctestify)**:  
```bash
grpctestify examples/projects/sensor-monitor/
```  

## 📡 IoT Monitoring Patterns  
This example shows you how to build IoT monitoring systems in practice:  
- **Real-Time Data Streaming**: Continuous sensor data flow without interruption - like a live feed from security cameras  
- **IoT Device Integration**: Patterns for connecting multiple sensor devices - because real IoT systems have dozens of sensors  
- **Time-Series Processing**: Handling chronological sensor data streams - useful for trend analysis and anomaly detection  
- **Sensor Data Formats**: Various sensor types (temperature, humidity, pressure, etc.) - different sensors produce different data  
- **Monitoring Dashboards**: Real-time data visualization patterns - what you'd see in a control room  
- **Alert Systems**: Threshold-based monitoring and alerting - like getting notified when temperature gets too high  

## ⚠️ Important Notes  
- Focus on **streaming behavior** and **real-time data processing**.  
- Demonstrates **sensor data monitoring** patterns.  
- Tests **continuous data flow** scenarios.  
- Ensure `gripmock` and `grpctestify` are installed (see their documentation for setup).  

## 🤝 Contributing  
Pull requests are welcome! Please ensure:  
- New tests cover **streaming scenarios**  
- Sensor data processing is properly tested  
- Real-time monitoring patterns are validated  

---

Made with ❤️ and protocol buffers 