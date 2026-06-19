<div align="center">
  <a href="https://alistgo.com"><img width="100px" alt="logo" src="https://cdn.jsdelivr.net/gh/alist-org/logo@main/logo.svg"/></a>
  <p><em>🗂一个支持多存储的文件列表程序 — API-only 模式，使用 Gin。</em></p>
<div>
  <a href="https://goreportcard.com/report/github.com/alist-org/alist/v3">
    <img src="https://goreportcard.com/badge/github.com/alist-org/alist/v3" alt="latest version" />
  </a>
  <a href="https://github.com/alist-org/alist/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/Xhofe/alist" alt="License" />
  </a>
  <a href="https://github.com/alist-org/alist/actions?query=workflow%3ABuild">
    <img src="https://img.shields.io/github/actions/workflow/status/Xhofe/alist/build.yml?branch=main" alt="Build status" />
  </a>
  <a href="https://github.com/alist-org/alist/releases">
    <img src="https://img.shields.io/github/release/Xhofe/alist" alt="latest version" />
  </a>
  <a title="Crowdin" target="_blank" href="https://crwd.in/alist">
    <img src="https://badges.crowdin.net/alist/localized.svg">
  </a>
</div>
<div>
  <a href="https://github.com/alist-org/alist/discussions">
    <img src="https://img.shields.io/github/discussions/Xhofe/alist?color=%23ED8936" alt="discussions" />
  </a>
  <a href="https://discord.gg/F4ymsH4xv2">
    <img src="https://img.shields.io/discord/1018870125102895134?logo=discord" alt="discussions" />
  </a>
  <a href="https://github.com/alist-org/alist/releases">
    <img src="https://img.shields.io/github/downloads/Xhofe/alist/total?color=%239F7AEA&logo=github" alt="Downloads" />
  </a>
  <a href="https://hub.docker.com/r/xhofe/alist">
    <img src="https://img.shields.io/docker/pulls/xhofe/alist?color=%2348BB78&logo=docker&label=pulls" alt="Downloads" />
  </a>
  <a href="https://alistgo.com/zh/guide/sponsor.html">
    <img src="https://img.shields.io/badge/%24-sponsor-F87171.svg" alt="sponsor" />
  </a>
</div>
</div>

---

[English](./README.md) | 中文 | [日本語](./README_ja.md) | [Contributing](./CONTRIBUTING.md) | [CODE_OF_CONDUCT](./CODE_OF_CONDUCT.md)

## 功能

- [x] 多种存储
    - [x] 本地存储
    - [x] [阿里云盘](https://www.alipan.com/)
    - [x] OneDrive / Sharepoint（[国际版](https://www.office.com/), [世纪互联](https://portal.partner.microsoftonline.cn),de,us）
    - [x] [天翼云盘](https://cloud.189.cn) (个人云, 家庭云)
    - [x] [GoogleDrive](https://drive.google.com/)
    - [x] [123云盘](https://www.123pan.com/)
    - [x] FTP / SFTP
    - [x] [PikPak](https://www.mypikpak.com/)
    - [x] [S3](https://aws.amazon.com/cn/s3/)
    - [x] [Seafile](https://seafile.com/)
    - [x] [又拍云对象存储](https://www.upyun.com/products/file-storage)
    - [x] WebDav(支持无API的OneDrive/SharePoint)
    - [x] Teambition（[中国](https://www.teambition.com/ )，[国际](https://us.teambition.com/ )）
    - [x] [MediaFire](https://www.mediafire.com)
    - [x] [分秒帧](https://www.mediatrack.cn/)
    - [x] [ProtonDrive](https://proton.me/drive)
    - [x] [和彩云](https://yun.139.com/) (个人云, 家庭云，共享群组)
    - [x] [Yandex.Disk](https://disk.yandex.com/)
    - [x] [百度网盘](http://pan.baidu.com/)
    - [x] [UC网盘](https://drive.uc.cn)
    - [x] [夸克网盘](https://pan.quark.cn)
    - [x] [迅雷网盘](https://pan.xunlei.com)
    - [x] [蓝奏云](https://www.lanzou.com/)
    - [x] [蓝奏云优享版](https://www.ilanzou.com/)
    - [x] [阿里云盘分享](https://www.alipan.com/)
    - [x] [谷歌相册](https://photos.google.com/)
    - [x] [Mega.nz](https://mega.nz)
    - [x] [一刻相册](https://photo.baidu.com/)
    - [x] SMB
    - [x] [115](https://115.com/)
    - [X] Cloudreve
    - [x] [Dropbox](https://www.dropbox.com/)
    - [x] [飞机盘](https://www.feijipan.com/)
    - [x] [多吉云](https://www.dogecloud.com/product/oss)
- [x] 部署方便，开箱即用
- [x] WebDav (具体见 https://alistgo.com/zh/guide/webdav.html)
- [x] [Docker 部署](https://hub.docker.com/r/xhofe/alist)
- [x] Cloudflare workers 中转
- [x] 跨存储复制文件
- [x] 单线程下载/串流的多线程下载加速
- [x] 纯 REST API 模式 — 前端由第三方客户端提供（如 OpenList、alist-web 等）。非 API 路由返回 JSON `404`。

## 文档

<https://alistgo.com/zh/>

## API 文档（通过 Apifox 提供）

<https://alist-public.apifox.cn/>

## Demo

公共演示站点（`https://al.nn.ci`）已随内嵌前端一同下线。使用 Docker（`xhofe/alist`）或源码构建部署后，使用任意兼容客户端连接即可。

## 讨论

一般问题请到[讨论论坛](https://github.com/alist-org/alist/discussions) ，**issue仅针对错误报告和功能请求。**

## 赞助

AList 是一个开源软件，如果你碰巧喜欢这个项目，并希望我继续下去，请考虑赞助我或提供一个单一的捐款！感谢所有的爱和支持：https://alistgo.com/zh/guide/sponsor.html

### 特别赞助

- [VidHub](https://apps.apple.com/app/apple-store/id1659622164?pt=118612019&ct=alist&mt=8) - 苹果生态下优雅的网盘视频播放器，iPhone，iPad，Mac，Apple TV全平台支持。

## 贡献者

Thanks goes to these wonderful people:

[![Contributors](http://contrib.nn.ci/api?repo=alist-org/alist&repo=alist-org/docs)](https://github.com/alist-org/alist/graphs/contributors)

## 许可

`AList` 是在 AGPL-3.0 许可下许可的开源软件。

## 免责声明
- 本程序为免费开源项目，旨在分享网盘文件，方便下载以及学习golang，使用时请遵守相关法律法规，请勿滥用；
- 本程序通过调用官方sdk/接口实现，无破坏官方接口行为；
- 本程序仅做302重定向/流量转发，不拦截、存储、篡改任何用户数据；
- 在使用本程序之前，你应了解并承担相应的风险，包括但不限于账号被ban，下载限速等，与本程序无关；
- 如有侵权，请通过[邮件](mailto:i@nn.ci)与我联系，会及时处理。

---

> [@博客](https://nn.ci/) · [@GitHub](https://github.com/alist-org) · [@Telegram群](https://t.me/alist_chat) · [@Discord](https://discord.gg/F4ymsH4xv2)
