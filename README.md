# 对白工坊 · DialogueForge 1.1

把 TXT 剧情转换成 Minecraft Java 1.21.1 的命令方块与红石中继器线路，输出 WorldEdit `.schem` 文件。

**[下载 Windows 软件包](https://github.com/mmhct/dialogue-import-mc/releases/latest/download/DialogueForge-1.1.0-Windows-x64.zip)** · [单独下载 EXE](https://github.com/mmhct/dialogue-import-mc/releases/latest/download/DialogueForge.exe) · [版本发布页](https://github.com/mmhct/dialogue-import-mc/releases)

游戏导入、播放顺序和中继器间隔已在 Paper 1.21.1 + WorldEdit 7.3.8 中实测。Windows EXE 已在 Wine 中运行验证，尚未在原生 Windows 电脑上实测。详细范围见 [验证记录](VALIDATION.md)。

## 直接使用

1. 解压发布包，在 Windows 64 位电脑上双击 `DialogueForge.exe`。
2. 选择或拖入多份 TXT，用 ↑ / ↓ 调整文件顺序，再点击“按顺序导入”。默认追加到当前剧情末尾，也可选择替换。
3. 在表格中直接编辑对白或指令，统一或逐条调整间隔。使用“传送、给予、标题、效果、自定义指令”按钮插入条目。
4. 设置“最大宽度”和“掉头间距”，选择全部合并、按条数、按来源文件或手动分块。
5. 预览每个结构后导出。单个结构生成 `.schem`，多个结构打包成 ZIP，也可仅导出当前结构。

EXE 已内置程序和界面，不需要安装 Python、Java 或 Node.js，使用过程无需联网。浏览器中的地址以 `http://127.0.0.1:` 开头，文本只在本机处理。Minecraft 本身所需的 Java 和模组环境另计。

点击右上角“退出软件”可结束程序。仅关闭浏览器标签页不会结束后台程序。剧情编辑没有自动保存，关闭前点击“保存项目”；下次通过“载入项目”继续编辑。项目保留文本、指令、间隔、来源、分界和线路设置，兼容 1.0 的设置文件（默认使用 3 格间距）。

## TXT 格式

```text
# 第一幕：备注不会进入游戏
A：别出声，前面有人。
B：你确定是这条路吗？
/title @a title {"text":"遗迹入口","color":"gold"}
A：拿好钻石，跟我来。
/give @a minecraft:diamond 1
A：看见那扇门了吗？
```

- 每个实际换行分出一个条目，每条对白或指令对应一个命令方块。
- **行首为 `#`：无条件跳过整行**，不生成方块、不占用间隔，也不计入分块条数。保留后续条目的原始行号。
- **行首为 `/`：作为原始指令**，去掉首个 `/` 后直接写入命令方块，不做 `tellraw` 包裹。
- 其余行作为对白。判断的是第一个字符；前面有空格的 ` #备注` 或 ` /指令` 会按对白处理。
- 连续两行都是 A，也会生成两个命令方块，不会合并。
- 不解析角色名：整行按原文播报，包括 `A：`、标点和空格。
- 编辑器的自动折行不算换行；按 Enter 产生的换行才算。
- 默认忽略空行。取消“忽略空行”后，中间空行也会占一个命令方块和一个间隔。
- 文件末尾的一个换行符只是结束最后一行，不会额外生成空白对白。
- 推荐 UTF-8；界面也支持 GB18030 / GBK 和 UTF-16，可手动选择编码。

生成的命令类似 `tellraw @a {"text":"A：别出声，前面有人。"}`。软件会自动处理 JSON 引号、反斜杠等转义。默认 `@a` 向所有玩家发送，可将播报对象改为 `@p` 或带条件的玩家选择器。选择器条件必须符合 1.21.1 的命令语法；软件不代替游戏完整检查条件。普通命令方块没有实体执行者，使用 `@s` 通常不会找到玩家。

## 插入指令和编辑

点击一条记录选中它，再选择插入位置：选中条目前、选中条目后或剧情末尾。点击“传送”会插入 `/tp @a `，随后填写坐标，例如 `/tp @a 100 64 200`。给予、标题和效果模板带有可修改的示例参数。

条目类型可在“对白 / 指令”下拉框中切换，此选择优先于内容的 `/` 前缀。输入框中的实际换行会被拦截，请新增条目来放下一句。

原始指令的执行上下文是该命令方块：`~` 相对它的位置，`@s` 默认没有实体执行者。软件保存指令原文并检查单行格式，不替代 Minecraft 的完整语法检查；请补全模板参数。指令与对白共享红石线路和间隔设置。

## 合并与分块

- **全部合并**：按文件队列和表格顺序，将所有 TXT 的有效条目连成一个结构，文件边界不会额外停顿。
- **按条数**：每 N 条生成一个结构，末块保留余下条目。对白与指令都计数。
- **按来源文件**：每段连续的来源文件独立成块；重复导入同名文件仍视为不同来源。手动插入条目继承相邻条目的来源。跨来源移动条目可能增加分段。
- **手动分界**：点击条目的“分块”，从此条起新建一块；再次点击取消。由自动分块切换时保留当前分界。

每个结构都有独立按钮，第一条从零开始计时。**块之间不会自动接线或连续播放，每块最后一条的间隔不生效。** 通过“预览结构”切换查看，点击表格条目可定位到所在块。

ZIP 内含编号 `.schem`、`manifest.json`（条目范围、尺寸、时长）和 `project.dialogue.json`（完整可编辑项目）。请解压 `.schem` 后分别放入 WorldEdit 目录。

## 时间、宽度和掉头间距

“条目后间隔”表示当前对白或指令执行到下一条执行的间隔，每块最后一条不再追加延时。

- 每句间隔支持 **0.1～600 秒**，精度 **0.1 秒**。
- 每个中继器可贡献 0.1～0.4 秒。软件优先使用 0.4 秒满档，末个中继器自动补足余量。
- 例如 2 秒使用 5 个满档中继器；2.3 秒使用 5 个满档中继器和 1 个 0.3 秒中继器。
- 时间按正常 **20 TPS** 计算。服务器掉 TPS 时，实际等待时间会变长。
- 最大宽度支持 **8～256 格**，包含按钮、转弯和边缘。实际生成宽度可能小于设置值，预览显示实际尺寸。
- 掉头间距是两条水平主线之间空出的格数，默认 **3**，可设为 **大于等于 1 的整数**，不能为 0。例如间距 1 时相邻主线 Z 坐标差为 2；间距 3 时差为 4。受结构格式和总体积约束，设置上限为 32758。
- 线路到达宽度上限后沿 Z 方向掉头。转角使用红石粉，不额外增加中继器延时。
- 结构高度固定 3 格：玻璃隔离底座与命令方块、红石线路、顶部空气层。线路可能很长。

一个项目最多 10000 条有效条目，每个结构最多 2000 条，一次最多导出 256 块。单块最多 200 万格、整批最多 1000 万格，同时受线路元件数限制。TXT 每份最多 4 MB，项目/API 请求最多 8 MB；超过限制时会显示原因。编辑表格按每页 100 条显示。

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
- `example.dialogue.json`：上述示例的设置，可通过“载入项目”打开。
- `混合剧情.txt`、`混合剧情.dialogue.json`：包含注释、指令和按条数分块的新版示例。
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

批量与分块示例：

```powershell
& .\DialogueForge.exe --input chapter1.txt --input chapter2.txt --gap 1 --width 32 --output merged.schem | Out-String
& .\DialogueForge.exe --input story.txt --split-mode count --split-every 50 --output chapters.zip | Out-String
& .\DialogueForge.exe --input chapter1.txt --input chapter2.txt --split-mode files --output chapters.zip | Out-String
```

`--input` 可重复，按参数顺序导入；`--gap` 设置空格数；`--split-mode` 可选 `none/count/files/manual`，手动分界从项目文件读取。一个结构要求 `.schem` 扩展名，多个结构要求 `.zip`；省略 `--output` 自动命名。`--project` 使用项目内设置，与 `--input` 不可同时指定。

命令行 TXT 支持 UTF-8 及带 BOM 的 UTF-16；GBK 请使用界面导入。`--project 文件.dialogue.json` 可恢复所有编辑设置，`--preview-json layout.json` 导出第一个结构的布局及所有分块的范围、尺寸和时长。

源码目录：`internal/circuit` 负责布局与计时，`internal/project` 管理合并、分块和 ZIP，`internal/schematic` 写结构文件，`internal/textfile` 读取文本，`internal/app` 提供本机界面服务，`web` 为界面资源。`scripts/game_validation.cjs` 是隔离测试服务器的导入和播放验证程序，不是日常使用依赖。
