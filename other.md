# wepoc

## Nuclei 漏洞扫描器图形界面工具

[![wepoc Logo](https://img.shields.io/badge/wepoc-Nuclei%20GUI-2E8B57?style=for-the-badge&logo=shield&logoColor=white)](https://github.com/cyber0s/wepoc)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=white)](https://reactjs.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-FF6B6B?style=for-the-badge&logo=wails&logoColor=white)](https://wails.io/)
[![License](https://img.shields.io/badge/License-GPL--3.0-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white)](LICENSE)

> 基于 Wails v2 框架的 Nuclei 漏洞扫描器图形界面工具  
> 专为安全研究人员和渗透测试工程师设计

---

## 项目简介

**wepoc** 是一个专为安全研究人员和渗透测试工程师设计的现代化漏洞扫描工具。

基于强大的 [Nuclei](https://github.com/projectdiscovery/nuclei) 扫描引擎，通过图形界面让漏洞扫描变得更加简单高效。

目前发布了 m 系列芯片的，后续发布交叉编译的。

## 功能特性

### 核心功能一览

### 模板管理

- 批量导入 Nuclei YAML 模板
- 智能验证和去重处理
- 高级筛选按关键词和严重等级
- 搜索分类模板管理

![模板管理](https://free.picui.cn/free/2025/10/22/68f895d800cf7.png)

![模板管理界面](https://free.picui.cn/free/2025/10/22/68f895d8282f1.png)

![模板筛选](https://free.picui.cn/free/2025/10/22/68f895d8be3b3.png)

![模板详情](https://free.picui.cn/free/2025/10/22/68f895d8b292f.png)

### 扫描任务

- 多任务并发异步处理，多任务之间不会相互影响
- 实时监控进度跟踪，发现漏洞右上角弹框提示且全局生效
- 任务重扫支持重新扫描
- 状态保持多选POC模板

![扫描任务](https://free.picui.cn/free/2025/10/22/68f895da37d7b.png)

![扫描任务详情](https://free.picui.cn/free/2025/10/22/68f895db39e72.png)

![模板管理详情](https://free.picui.cn/free/2025/10/22/68f895dd98602.png)

![扫描任务配置](https://free.picui.cn/free/2025/10/22/68f895ddb084b.png)

![扫描任务状态](https://free.picui.cn/free/2025/10/22/68f895de0afa1.png)

![扫描进度](https://free.picui.cn/free/2025/10/22/68f895d8db207.png)

![扫描结果](https://free.picui.cn/free/2025/10/22/68f895de70636.png)

### 结果分析

- 详细展示漏洞信息
- 请求响应数据查看请求包，响应包
- 实时通知扫描结果右上角抽屉弹框提示

![结果分析](https://free.picui.cn/free/2025/10/22/68f895e0436ff.png)

![配置管理详情](https://free.picui.cn/free/2025/10/22/68f895e11b278.png)

### 配置管理

- POC导入增量导入，自动存储到~/.wepoc/nuclei-templates 目录
- 配置保存持久化设置相关配置文件，全部都在 ~/.wepoc 目录

![配置管理](https://free.picui.cn/free/2025/10/22/68f895d800cf7.png)

![系统设置](https://free.picui.cn/free/2025/10/22/68f895d8282f1.png)

## 许可证

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white)](LICENSE)

本项目采用 [GPL-3.0 License](LICENSE) 许可证。

**开发者不对使用本工具可能产生的任何法律后果承担责任。**

## 致谢

### [Nuclei](https://github.com/projectdiscovery/nuclei)

### [Wails](https://wails.io/)

### [Ant Design](https://ant.design/)

### [React](https://reactjs.org/)

---

**如果这个项目对你有帮助，请给我们一个 Star！**

[![GitHub stars](https://img.shields.io/github/stars/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)
[![GitHub forks](https://img.shields.io/github/forks/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)
