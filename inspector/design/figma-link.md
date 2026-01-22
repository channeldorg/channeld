# Channeld Inspector UI 设计

## Figma链接
- [频道列表模式] https://www.figma.com/design/yrGJWmR0ieuHN7fSfnNpD9/ChanneldInspector?node-id=0-1&t=t4F2Ud54ijuSjOSy-1

## 设计规范

### 布局结构
- **整体背景**: 黑色
- **左侧频道列表**: 360px宽，灰色背景 (#ccc)
- **标题栏**: 顶部，56px高
- **标签页区域**: 50px高
- **频道数据区域**: 透明背景，白色边框

### 交互设计

#### 频道数据展示
- **展示方式**: JSON树（Tree View）
- **交互功能**: 每个叶属性（leaf property）都可以点击进行修改
- **实现要求**: 
  - 支持展开/折叠节点
  - 支持内联编辑
  - 支持数据类型验证
  - 修改后可以保存

### 模式切换
- 频道列表模式
- 空间频道模式
- 连接视图模式

### 待确认的设计细节
- 颜色系统（除黑色背景和灰色侧边栏外的主色调）
- 字体和字号规范
- 间距系统（8px/16px基础单位等）
- 交互状态（hover、active等）
