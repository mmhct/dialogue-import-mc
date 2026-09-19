# 对白工坊 · DialogueForge 1.0

把 TXT 剧情转换成 Minecraft Java 1.21.1 的命令方块与红石中继器线路，输出 WorldEdit `.schem` 文件。

**[下载 Windows 软件包](https://github.com/mmhct/dialogue-import-mc/releases/latest/download/DialogueForge-1.0.0-Windows-x64.zip)** · [单独下载 EXE](https://github.com/mmhct/dialogue-import-mc/releases/latest/download/DialogueForge.exe) · [版本发布页](https://github.com/mmhct/dialogue-import-mc/releases)

游戏导入、播放顺序和中继器间隔已在 Paper 1.21.1 + WorldEdit 7.3.8 中实测。Windows EXE 已在 Wine 中运行验证，尚未在原生 Windows 电脑上实测。详细范围见 [验证记录](VALIDATION.md)。

## 直接使用

1. 解压发布包，在 Windows 64 位电脑上双击 `DialogueForge.exe`。
2. 软件会打开默认浏览器中的本机界面。点击“选择文件”导入 TXT，也可以拖入文件或试用示例。
3. 设置默认间隔并点击“应用到所有句”，或在表格中逐句修改“本句后间隔”。
4. 设置“最大宽度”，在预览中检查自动掉头的线路。
5. 点击“导出结构文件”，浏览器会下载 `.schem` 文件。

EXE 已内置程序和界面，不需要安装 Python、Java 或 Node.js，使用过程无需联网。浏览器中的地址以 `http://127.0.0.1:` 开头，文本只在本机处理。Minecraft 本身所需的 Java 和模组环境另计。

点击右上角“退出软件”可结束程序。仅关闭浏览器标签页不会结束后台程序。逐句间隔没有自动保存，关闭前可点击“保存设置”；下次通过“载入设置”继续编辑。

## TXT 格式

```text
A：别出声，前面有人。
B：你确定是这条路吗？
A：看见那扇门了吗？
A：等我发出信号，我们一起进去。
```

- 每个实际换行分出一条对白，每条对应一个命令方块。
- 连续两行都是 A，也会生成两个命令方块，不会合并。
- 不解析角色名：整行按原文播报，包括 `A：`、标点和空格。
- 编辑器的自动折行不算换行；按 Enter 产生的换行才算。
- 默认忽略空行。取消“忽略空行”后，中间空行也会占一个命令方块和一个间隔。
- 文件末尾的一个换行符只是结束最后一行，不会额外生成空白对白。
- 推荐 UTF-8；界面也支持 GB18030 / GBK 和 UTF-16，可手动选择编码。

生成的命令类似 `tellraw @a {"text":"A：别出声，前面有人。"}`。软件会自动处理 JSON 引号、反斜杠等转义。默认 `@a` 向所有玩家发送，可将播报对象改为 `@p` 或带条件的玩家选择器。选择器条件必须符合 1.21.1 的命令语法；软件不代替游戏完整检查条件。普通命令方块没有实体执行者，使用 `@s` 通常不会找到玩家。

## 时间和宽度

“本句后间隔”表示这句输出到下一句输出的间隔，最后一句不再追加延时。

- 每句间隔支持 **0.1～600 秒**，精度 **0.1 秒**。
- 每个中继器可贡献 0.1～0.4 秒。软件优先使用 0.4 秒满档，末个中继器自动补足余量。
- 例如 2 秒使用 5 个满档中继器；2.3 秒使用 5 个满档中继器和 1 个 0.3 秒中继器。
- 时间按正常 **20 TPS** 计算。服务器掉 TPS 时，实际等待时间会变长。
- 最大宽度支持 **8～256 格**，包含按钮、转弯和边缘。实际生成宽度可能小于设置值，预览显示实际尺寸。
- 线路到达宽度上限后沿 Z 方向掉头。转角使用红石粉，不额外增加中继器延时。
- 结构高度固定 3 格：玻璃隔离底座与命令方块、红石线路、顶部空气层。线路可能很长。

单次最多 2000 条对白，同时受线路元件数与结构体积限制。超过限制时会显示原因，建议按场景拆分导出。

## 导入 Minecraft Java 1.21.1

需要安装支持 1.21.1 的 WorldEdit。此软件生成 **Sponge Schematic v3 `.schem`**，供 WorldEdit 导入；它不是原版结构方块使用的 `.nbt`，也不是 Litematica 的 `.litematic`。

WorldEdit 作者发布页：<https://modrinth.com/plugin/worldedit/versions>。请选择对应 Minecraft 版本及模组加载器/服务器类型的文件。

1. 将导出的 `.schem` 放入实际运行游戏或服务器的 WorldEdit `schematics` 目录。常见位置为：
   - Paper / Spigot 服务器：服务器目录下 `plugins/WorldEdit/schematics/`。
   - Fabric / NeoForge 单人环境：对应游戏实例下 `config/worldedit/schematics/`。以该实例的 WorldEdit 配置为准。
2. 如果连接远程服务器，应把文件放到服务器端。仅放到自己电脑的客户端目录不会被服务器读取。
3. 进入创造模式并拥有 WorldEdit/命令方块权限。在空旷位置执行：

   ```text
   //schem load example
   //paste
   ```

   发布包的示例结构名为 `example.schem`；其他文件请替换文件名。
4. 金色方块是起点。右键它上面的石按钮，开始播放对白。

默认粘贴以玩家所在方块为底座起点，线路向 +X 延伸并向 +Z 折返。请给结构留出空间，保留顶部空气层，使用普通 `//paste`。服务器在 `server.properties` 中需要设置 `enable-command-block=true` 并重启。需要时用 `/gamerule commandBlockOutput false` 隐藏命令方块的额外反馈。

播放过程中，**整条线路所在区块必须持续加载**。较长线路可由管理员根据实际坐标使用原版 `/forceload`，或放在已经保持加载的区域。软件不会擅自修改游戏的区块加载规则。

同一条线路播放完后，再等约 1 秒让信号完全复位，即可重播。播放途中再次按按钮可能叠加两轮对白；当前版本没有取消、暂停、分支或防重复启动功能。

线路导入后由原版命令方块和红石运行，不需要让本软件保持打开。

WorldEdit 导入说明：<https://worldedit.enginehub.org/en/latest/usage/clipboard/>。

## 发布包内容

- `DialogueForge.exe`：Windows x86-64 软件。
- `使用说明.txt`：本说明。
- `示例剧情.txt`：可直接导入的软件示例。
- `example.schem`：6 条对白、每句间隔 2 秒、最大宽度 8 格的示例结构。
- `example.dialogue.json`：上述示例的设置，可通过“载入设置”打开。
- `验证记录.md`：实际测试范围及未验证项。
- `SHA256SUMS.txt`：文件校验值。

## 从源码构建

使用 Go 1.24 或更高版本，无第三方 Go 包依赖。

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
```

或在 macOS / Linux 上：

```sh
sh scripts/build.sh
```

构建结果位于 `dist/`。Windows 版本内置完整网页资源，没有运行时 CDN 或在线 API 依赖。

开发运行：`go run ./cmd/dialogueforge`。检查：`go test -race ./...`、`go vet ./...`。

也支持命令行转换，Windows GUI 版建议在 PowerShell 中等待程序结束：

```powershell
& .\DialogueForge.exe --input '.\示例剧情.txt' --output '.\example.schem' --width 8 --interval 2 | Out-String
```

命令行 TXT 支持 UTF-8 及带 BOM 的 UTF-16；GBK 请使用界面导入。`--project 文件.dialogue.json` 可恢复各句不同间隔，`--preview-json layout.json` 可导出布局和时间信息。高级参数可从源码入口查看。

源码目录：`internal/circuit` 负责布局与计时，`internal/schematic` 写结构文件，`internal/textfile` 读取文本，`internal/app` 提供本机界面服务，`web` 为界面资源。`scripts/game_validation.cjs` 是隔离测试服务器的导入和播放验证程序，不是日常使用依赖。
