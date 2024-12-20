# GoDown

Multithread downloader driven by Go

![WindowsTerminal_WvJe9OqoNa](https://github.com/user-attachments/assets/0c1cbb22-a289-48d1-8bc0-0d6f456f56d4)
[[commit d24601]](github.com/Miuzarte/GoDown/commit/d2460173ffa7a4cda86cbabf0500fe198ca6646b) 对线程进度条增加了优先级(取块索引), 进度条每次更新不再会将自身置顶, GIF懒得重录了

## Features

- **Download in parallel but write sequentially, HDD friendly**
- Auto identify downloads folder (Windows only)
- Fancy and useless progress bar
- Output path as a hyperlink
