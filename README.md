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

### 🌟 主要特点

- 🎨 **现代化界面**: 基于 React 18 + Ant Design 5 构建的直观 UI
- ⚡ **高性能**: 采用 Wails v2 框架，原生性能体验
- 🔄 **实时反馈**: 扫描过程实时显示，漏洞即时通知
- 💾 **数据持久化**: 所有配置和结果自动保存
- 🛠️ **POC 编辑器**: 内置 YAML 编辑器，支持在线测试 POC
- 📊 **结果导出**: 支持 CSV 格式导出扫描结果

</div>

## 📦 下载安装

### 📥 预编译版本

从 [Releases](https://github.com/cyber0s/wepoc/releases) 页面下载对应平台的可执行文件：

- **macOS**: `wepoc-macos-universal.app` (支持 Intel 和 Apple Silicon)
- **Windows**: `wepoc-windows-amd64.exe`

### 🔧 前置要求

**必须先安装 Nuclei 扫描引擎**，否则无法使用扫描功能。

#### macOS / Linux

```bash
# 使用 Go 安装
go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

# 或使用 Homebrew (macOS)
brew install nuclei
```

#### Windows

```powershell
# 使用 Go 安装
go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

# 或下载预编译版本
# 访问 https://github.com/projectdiscovery/nuclei/releases
```

### 🚀 启动应用

1. **macOS**: 双击 `wepoc.app`
   - 首次运行可能提示"无法验证开发者"，请前往"系统偏好设置 > 安全性与隐私"允许运行

2. **Windows**: 双击 `wepoc.exe`
   - 可能提示 Windows Defender 警告，选择"仍要运行"

3. **首次使用**:
   - 阅读并同意《网络安全法》使用协议
   - 在"设置"页面配置 Nuclei 可执行文件路径
   - 导入 POC 模板（从本地目录或官方模板库）

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


## 📖 使用指南

### 1️⃣ 导入 POC 模板

1. 进入"模板管理"页面
2. 点击"导入模板"按钮
3. 选择包含 Nuclei YAML 文件的目录
4. 系统会自动验证、去重并导入有效模板
5. 导入的模板存储在 `~/.wepoc/nuclei-templates/` 目录

**推荐模板库**:
- [Nuclei 官方模板](https://github.com/projectdiscovery/nuclei-templates)
- [POC-in-GitHub](https://github.com/topics/nuclei-templates)

### 2️⃣ 创建扫描任务

1. 进入"扫描任务"页面
2. 点击"新建任务"
3. 配置扫描参数：
   - **任务名称**: 自定义任务标识
   - **选择 POC**: 支持多选模板（按 Severity 筛选）
   - **目标列表**: 每行一个目标（支持 URL、IP、域名）
   - **高级选项**: 并发数、速率限制、代理设置等
4. 点击"开始扫描"

### 3️⃣ 查看扫描结果

1. 实时进度显示在任务列表
2. 发现漏洞时右上角弹窗提示（全局生效）
3. 点击"查看详情"进入结果分析页面
4. 查看：
   - 漏洞详情（CVE、严重程度、描述）
   - HTTP 请求/响应包
   - 提取的数据（如版本号、敏感信息）
5. 支持导出 CSV 报告

### 4️⃣ POC 编辑与测试

1. 在任务详情页点击漏洞的"测试 POC"按钮
2. 进入 POC 编辑器界面：
   - **左侧**: YAML 内容编辑区（语法高亮）
   - **右侧**: 测试参数配置区
3. 配置测试参数（目标 URL、并发、代理等）
4. 点击"测试 POC"查看实时结果
5. 修改后点击"保存"（自动创建备份）

## ❓ 常见问题 (FAQ)

### Q1: 提示"找不到 Nuclei 可执行文件"？

**解决方案**:
1. 确认已安装 Nuclei: `nuclei -version`
2. 在"设置"页面手动选择 Nuclei 路径
3. macOS 通常在: `/usr/local/bin/nuclei` 或 `~/go/bin/nuclei`
4. Windows 通常在: `C:\Users\用户名\go\bin\nuclei.exe`

### Q2: macOS 提示"无法打开应用，因为它来自身份不明的开发者"？

**解决方案**:
```bash
# 终端执行（替换为实际路径）
xattr -cr /Applications/wepoc.app

# 或前往"系统偏好设置 > 安全性与隐私 > 通用"允许运行
```

### Q3: 扫描失败，显示 "exit status 2"？

**可能原因**:
- Nuclei 参数不兼容（版本问题）
- 目标 URL 格式错误（缺少 `http://` 前缀）
- POC 模板语法错误

**解决方案**:
1. 更新 Nuclei 到最新版本
2. 检查目标格式（系统会自动补全协议）
3. 使用 POC 编辑器测试单个模板

### Q4: 如何配置代理？

1. 进入"设置"页面
2. 在"网络代理"栏填写代理地址
3. 支持格式:
   - HTTP: `http://127.0.0.1:8080`
   - HTTPS: `https://127.0.0.1:8080`
   - SOCKS5: `socks5://127.0.0.1:1080`
4. 点击"测试代理"验证连通性

### Q5: 数据存储在哪里？

所有数据存储在 `~/.wepoc/` 目录：
```
~/.wepoc/
├── config.json              # 应用配置
├── nuclei-templates/        # POC 模板
├── tasks/                   # 任务配置文件
├── results/                 # 扫描结果 (JSON)
└── logs/                    # 任务日志
```

## 🔨 从源码构建

### 环境要求

- Go 1.24+
- Node.js 18+
- Wails v2 CLI

### 构建步骤

```bash
# 1. 克隆仓库
git clone https://github.com/cyber0s/wepoc.git
cd wepoc

# 2. 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 3. 安装前端依赖
cd frontend
npm install
cd ..

# 4. 开发模式运行
wails dev

# 5. 构建生产版本
wails build

# 6. 跨平台编译
wails build -platform darwin/universal  # macOS Universal
wails build -platform windows/amd64     # Windows 64位
```

构建后的文件位于 `build/bin/` 目录。

## 🛡️ 免责声明

**重要提示**:

本工具仅供授权的安全测试使用，使用者必须遵守：

- ✅ 仅在获得授权的系统上使用
- ✅ 用于 CTF 竞赛、安全研究、漏洞复现
- ✅ 企业内部安全审计和渗透测试
- ❌ 不得用于未授权的网络攻击
- ❌ 不得用于非法获取他人信息
- ❌ 不得用于破坏计算机信息系统

**法律责任**:

根据《中华人民共和国网络安全法》《中华人民共和国刑法》等相关法律法规，未经授权对他人系统进行扫描、入侵等行为可能构成犯罪。使用本工具进行任何违法活动，后果由使用者自行承担，开发者不承担任何法律责任。

## 🗓️ 开发计划

- [ ] 🔐 代理池支持
- [ ] ☁️ 云端 POC 模板更新
- [ ] ✍️ 可视化 POC 创建器
- [ ] 📧 邮件/Webhook 通知
- [ ] 🌐 分布式扫描支持
- [ ] 📱 移动端适配
- [ ] 🐛 持续修复 Bug

## 📄 许可证

<div align="center">

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white)](LICENSE)

本项目采用 [GPL-3.0 License](LICENSE) 许可证。

</div>


**开发者不对使用本工具可能产生的任何法律后果承担责任。**

## 🏗️ 技术架构

### 核心技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| **后端** | Go | 1.24+ |
| **框架** | Wails | v2.10+ |
| **前端** | React | 18.2+ |
| **UI 库** | Ant Design | 5.x |
| **构建工具** | Vite | 5.x |
| **语言** | TypeScript | 5.x |
| **数据库** | SQLite | modernc.org/sqlite |
| **扫描引擎** | Nuclei | 外部可执行文件 |

### 项目结构

```
wepoc/
├── app.go                      # 主应用逻辑和前端 API 绑定
├── main.go                     # Wails 应用入口
├── internal/
│   ├── config/                 # 配置管理
│   ├── database/               # SQLite 数据库操作
│   ├── models/                 # 数据模型定义
│   └── scanner/                # Nuclei 扫描器封装
│       ├── nuclei.go           # Nuclei 执行逻辑
│       ├── template_parser.go  # POC 模板解析
│       ├── json_task_manager.go # JSON 任务管理器
│       └── enhanced_logger.go  # 增强日志解析
├── frontend/
│   ├── src/
│   │   ├── components/         # React 组件
│   │   ├── pages/              # 页面组件
│   │   │   ├── Templates/      # 模板管理
│   │   │   ├── ScanTasks/      # 扫描任务
│   │   │   ├── Results/        # 结果分析
│   │   │   └── Settings/       # 系统设置
│   │   ├── services/           # API 封装
│   │   └── types/              # TypeScript 类型定义
│   └── wailsjs/                # Wails 自动生成的绑定
├── build/                      # 构建资源和图标
└── wails.json                  # Wails 项目配置
```

### 数据流设计

```
前端 UI (React)
    ↓ 调用 Wails Binding
后端 API (app.go)
    ↓ 调用 Scanner
Nuclei 引擎执行
    ↓ 实时输出
Enhanced Logger 解析
    ↓ 发送 Event
前端监听更新 UI
```

## 🤝 贡献指南

欢迎所有形式的贡献！无论是报告 Bug、提出新功能建议，还是提交代码改进。

### 报告问题

在 [Issues](https://github.com/cyber0s/wepoc/issues) 页面提交 Bug 或功能请求时，请包含：

- **Bug 报告**:
  - 操作系统和版本
  - wepoc 版本
  - Nuclei 版本
  - 复现步骤
  - 错误日志（如有）

- **功能请求**:
  - 功能描述
  - 使用场景
  - 预期行为

### 代码贡献

1. Fork 本仓库
2. 创建特性分支: `git checkout -b feature/your-feature`
3. 提交更改: `git commit -m 'Add some feature'`
4. 推送到分支: `git push origin feature/your-feature`
5. 提交 Pull Request

**开发规范**:
- Go 代码遵循 `go fmt` 和 `golint` 规范
- TypeScript 代码使用 ESLint
- 提交信息使用语义化格式
- 新功能需添加对应测试（如适用）

### 本地开发环境搭建

```bash
# 克隆仓库
git clone https://github.com/cyber0s/wepoc.git
cd wepoc

# 安装依赖
cd frontend && npm install && cd ..

# 启动开发服务器（热重载）
wails dev

# 运行测试
go test ./...
cd frontend && npm test
```

## 🙏 致谢

<div align="center">


🔍 [Nuclei](https://github.com/projectdiscovery/nuclei)

🖥️ [Wails](https://wails.io/)

 🎨 [Ant Design](https://ant.design/)
 
 ⚛️ [React](https://reactjs.org/)
</div>


---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！**

[![GitHub stars](https://img.shields.io/github/stars/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)
[![GitHub forks](https://img.shields.io/github/forks/cyber0s/wepoc?style=social)](https://github.com/cyber0s/wepoc)

</div>
