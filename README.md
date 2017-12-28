# MailHide Mirror
A mirror service of [reCAPTCHA Mailhide](https://www.google.com/recaptcha/admin#mailhide) based on [App Engine](https://cloud.google.com/appengine/). It stores emails in [Google Cloud Datastore](https://cloud.google.com/datastore/) calls the [reCAPTCHA V2 API](https://developers.google.com/recaptcha/intro) to hide user's email address from malicious users. User has to pass the "I'm not a Robot" test to view an email address.

[reCAPTCHA Mailhide](https://www.google.com/recaptcha/admin#mailhide) 的镜像服务，基于 [App Engine](https://cloud.google.com/appengine/) 和 [Google Cloud Datastore](https://cloud.google.com/datastore/)，通过reCAPTCHA V2 来隐藏用户电子邮箱地址，免受垃圾邮件烦扰。用户需通过“我不是机器人”按钮才可查看邮箱地址。


# Hosted Version
## 关于

Users in Mainland China can access our hosted version at [https://mailhide.cn](https://mailhide.cn). This version is provided to GDG communities for free, with servers sponsored by [Google](https://www.google.com). We authorize all non-profit organizations, open-source communities, and technical groups to use it.

可在中国大陆无障碍访问的位于 [https://mailhide.cn](https://mailhide.cn) 的服务是为 GDG 社区免费提供的，服务器资源由 [Google](https://www.google.com) 赞助，我们授权所有非盈利组织、开源社区、技术社区使用。

 

## 隐私声明
* 我们珍视你给予我们的信任，我们一定不会辜负这份信任
* 我们防止垃圾邮件，我们不会给您发送任何邮件，也不会将您的邮箱地址分享给他人
* 我们希望复杂的网络环境下，能有一些最基本的信任，如有任何疑虑，请就自行搭建服务器

# Self-Host Tutorial
*TBA*

# Contribution
All submissions, including submissions by project members, require review. We use Github pull requests for this purpose. Thank you in advance!

# License
[MIT](LICENSE)
