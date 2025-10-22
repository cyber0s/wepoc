<div align="center">

# 🛡️ wepoc

### Nuclei 漏洞扫描器图形界面工具

[![wepoc Logo](https://img.shields.io/badge/wepoc-Nuclei%20GUI-2E8B57?style=for-the-badge&logo=shield&logoColor=white)](https://github.com/cyber0s/wepoc)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=white)](https://reactjs.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-FF6B6B?style=for-the-badge&logo=wails&logoColor=white)](https://wails.io/)
[![License](https://img.shields.io/badge/License-GPL--3.0-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white)](LICENSE)

> 🚀 **基于 Wails v2 框架的 Nuclei 漏洞扫描器图形界面工具**  
> 专为安全研究人员和渗透测试工程师设计

---

</div>

## 📖 项目简介

<div align="">

**wepoc** 是一个专为安全研究人员和渗透测试工程师设计的现代化漏洞扫描工具。

基于强大的 [Nuclei](https://github.com/projectdiscovery/nuclei) 扫描引擎，通过图形界面让漏洞扫描变得更加简单高效。

目前发布了 m 系列芯片的，后续发布交叉编译的。

</div>

## ✨ 功能特性

<div align="">

### 🔍 核心功能一览

</div>

### 📁 模板管理

- ✅ **批量导入** Nuclei YAML 模板
- ✅ **智能验证** 和去重处理
- ✅ **高级筛选** 按关键词和严重等级
- ✅ **搜索分类** 模板管理
-    <img src="./assets/image-20251021143608266.png" alt="模板管理" width="800"/>
-    <img width="800"  alt="image" src="https://github.com/user-attachments/assets/8b29cb30-d326-44ad-b917-b3c16c912267" />
-    <img width="800"  alt="image" src="https://github.com/user-attachments/assets/6d23499c-b86a-4b7f-ac91-6f9f628f5ca5" />
-    <img width="800"  alt="image" src="https://github.com/user-attachments/assets/8c65ac3d-71bb-42ec-a20a-ea746b6a9b40" />



### 🎯 扫描任务

- ✅ **多任务并发** 异步处理，多任务之间不会相互影响
- ✅ **实时监控** 进度跟踪，发现漏洞右上角弹框提示且全局生效
- ✅ **任务重扫** 支持重新扫描
- ✅ **状态保持** 多选POC模板
- <img src="./assets/image-20251021143658636.png" alt="扫描任务" width="800"/>
- <img src="./assets/image-20251021144128996.png" alt="扫描任务详情" width="800"/>
- <img src="./assets/image-20251021144048432.png" alt="模板管理详情" width="800"/>
- <img src="./assets/image-20251021144149421.png" alt="扫描任务配置" width="800"/>
- <img src="./assets/image-20251021144219739.png" alt="扫描任务状态" width="800"/>
  - <img src="./assets/image-20251021154345139.png" alt="扫描任务状态" width="800"/>
- <img src="./assets/image-20251021154420948.png" alt="扫描结果" width="800"/>

### 📊 结果分析

- ✅ **详细展示** 漏洞信息
- ✅ **请求响应** 数据查看 请求包，响应包
- ✅ **实时通知** 扫描结果 右上角抽屉弹框提示

- <img src="./assets/image-20251021154518940.png" alt="结果分析" width="800"/>

- <img src="./assets/image-20251021154628346.png" alt="配置管理详情" width="800"/>
- <img src="./assets/image-20251021154755624.png" alt="系统设置" width="800"/>
- <img width="800"  alt="image" src="https://github.com/user-attachments/assets/ab911d70-d31b-4ee1-8165-dcafc187e7ab" />
- <img width="800"  alt="image" src="https://github.com/user-attachments/assets/33bf4a22-6919-4eb7-98f5-658802c41b1c" />
- <img width="800"  alt="image" src="https://github.com/user-attachments/assets/4beb6eba-7ee1-43db-9319-d8c1100c3a70" />




### ⚙️ 配置管理

- ✅ **POC导入** 增量导入，自动存储到~/.wepoc/nuclei-templates 目录
- ✅ **配置保存** 持久化设置 相关配置文件，全部都在 ~/.wepoc 目录
- <img src="./assets/image-20251021154828421.png" alt="配置管理" width="800"/>

- <img width="800"  alt="image" src="https://github.com/user-attachments/assets/89d1bac4-df89-4df0-bbe1-dd50f760192e" />






## 📄 许可证

<div align="center">

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white)](LICENSE)

本项目采用 [GPL-3.0 License](LICENSE) 许可证。

</div>


**开发者不对使用本工具可能产生的任何法律后果承担责任。**

## 🙏 致谢

<div align="center">
</div>

### 🔍 [Nuclei](https://github.com/projectdiscovery/nuclei)

### 🖥️ [Wails](https://wails.io/)


### 🎨 [Ant Design](https://ant.design/)

### ⚛️ [React](https://reactjs.org/)



---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！**

[![GitHub stars](https://img.shields.io/github/stars/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)
[![GitHub forks](https://img.shields.io/github/forks/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)

</div>
