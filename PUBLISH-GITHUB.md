# 发布到 GitHub

建议仓库：`datout/UsbNetControl`

## 方式 A：GitHub CLI

在 Git Bash 中进入源码目录后：

```bash
git init -b main
git config user.name "datout"
git config user.email "minitruu@gmail.com"
git add .
git commit -m "Initial open source release v1.3.1"

gh auth login
gh repo create datout/UsbNetControl --public --source=. --remote=origin --push \
  --description "Windows GUI tool for USB storage access and network adapter control"

git tag v1.3.1
git push origin v1.3.1
```

推送 Tag 后，`.github/workflows/release.yml` 会自动构建 x64 / ARM64、生成 ZIP 与 SHA256，并创建 GitHub Release。

## 方式 B：已经在 GitHub 网页创建空仓库

如果已经创建了 `https://github.com/datout/UsbNetControl`：

```bash
git init -b main
git config user.name "datout"
git config user.email "minitruu@gmail.com"
git add .
git commit -m "Initial open source release v1.3.1"
git remote add origin https://github.com/datout/UsbNetControl.git
git push -u origin main

git tag v1.3.1
git push origin v1.3.1
```

## 后续发布

代码修改并提交后，只需要创建新 Tag，例如：

```bash
git tag v1.3.2
git push origin v1.3.2
```

Release Workflow 会根据 Tag 自动注入版本号并生成发布文件。
