# 对白工坊 DialogueForge v1.0.0

将每行 TXT 对白转换成 Minecraft Java 1.21.1 的命令方块，自动插入中继器，并按最大宽度蛇形折返。输出可由 WorldEdit 导入的 `.schem` 结构文件。

## 下载和使用

- **DialogueForge-1.0.0-Windows-x64.zip**：推荐下载，内含 Windows EXE、中文说明、示例 TXT、示例结构和设置文件。
- **DialogueForge.exe**：单独的 Windows x86-64 程序，已内置界面及运行所需资源。
- **DialogueForge-1.0.0-source.zip**：源代码归档。
- **SHA256SUMS.txt**：发布附件的 SHA-256 校验值。

解压后双击 EXE，软件会打开默认浏览器中的本机操作界面。导入 TXT，调整间隔和宽度，然后导出 `.schem`。通过 WorldEdit 导入后，按金色方块上的按钮播放。

## 功能

- 按实际换行分隔对白，每行一个命令方块，连续的同角色对白不会合并。
- 统一或逐句设置 0.1～600 秒间隔，精度 0.1 秒，由真实红石中继器控制。
- 最大宽度 8～256 格，超长自动掉头，提供线路预览。
- 默认使用 `tellraw @a`，可调整播报对象。
- 支持中文 TXT、保存和载入逐句设置；无需联网或安装其他运行库。

## 已验证与限制

已在 Paper 1.21.1 build 133 + WorldEdit 7.3.8 中完成 6 种布局、12 轮播放验证，覆盖多次折返、0.1 秒间隔和跨区块线路；正常 20 TPS 下另测两轮。对白顺序与命令方块实际执行间隔均通过检查。

Windows EXE 已在 Wine 11.17 中运行，导出结果与本机版本逐字节一致。**尚未在原生 Windows 电脑上实测。**

导出的文件为 WorldEdit `.schem`，不是原版结构方块 `.nbt`。游戏中需启用命令方块并保持整条线路所在区块加载；掉 TPS 会增加实际等待时间。当前版本不包含剧情分支、暂停或防重复启动功能。

[使用说明](https://github.com/mmhct/dialogue-import-mc/blob/main/README.md) · [完整验证记录](https://github.com/mmhct/dialogue-import-mc/blob/main/VALIDATION.md)
