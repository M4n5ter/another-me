FROM ubuntu:24.04 AS base

ENV DEBIAN_FRONTEND=noninteractive

# https://mirrors.tuna.tsinghua.edu.cn/help/ubuntu/
# https://mirrors.ustc.edu.cn/help/ubuntu.html
RUN sed -i 's@//.*archive.ubuntu.com@//mirrors.ustc.edu.cn@g' /etc/apt/sources.list.d/ubuntu.sources && \
    sed -i 's/security.ubuntu.com/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/ubuntu.sources
    # sed -i 's/http:/https:/g' /etc/apt/sources.list.d/ubuntu.sources
RUN apt-get update && apt-get install -y ca-certificates

RUN apt-get install -y\
    xfce4 \
    xfce4-goodies \
    dbus-x11 \
    tigervnc-standalone-server \
    tigervnc-common \
    chromium \
    fonts-wqy-zenhei \
    fonts-noto-cjk \
    sudo \
    net-tools \
    curl \
    vim

RUN apt-get clean && rm -rf /var/lib/apt/lists/* 

FROM base AS anotherme


RUN useradd -m -s /bin/bash anotherme && \
    echo "anotherme:yourpassword" | chpasswd && \
    adduser anotherme sudo

USER anotherme
WORKDIR /home/anotherme
RUN mkdir -p .vnc && \
    echo "yourpassword" | vncpasswd -f > .vnc/passwd && \
    chmod 600 .vnc/passwd

    RUN echo "#!/bin/bash\n\
    # 取消设置可能引起冲突的环境变量\n\
    unset SESSION_MANAGER\n\
    unset DBUS_SESSION_BUS_ADDRESS\n\
    \n\
    # 加载 X 资源 (可选)\n\
    if [ -r \$HOME/.Xresources ]; then\n\
        xrdb \$HOME/.Xresources\n\
    fi\n\
    \n\
    # 设置背景色 (可选)\n\
    xsetroot -solid grey\n\
    \n\
    # 启动 vncconfig 以支持剪贴板等 (后台运行)\n\
    vncconfig -iconic &\n\
    \n\
    # 启动 XFCE4 会话。使用 'exec' 来让 startxfce4 进程替换掉脚本的 shell 进程。\n\
    # 这是推荐的启动窗口管理器或桌面环境的方式。\n\
    exec startxfce4" > .vnc/xstartup && \
    chmod +x .vnc/xstartup

# 暴露 VNC 端口 (5900 + display number, 这里用 :1 即 5901)
EXPOSE 5901

CMD ["/usr/bin/vncserver", ":1", "-geometry", "1280x1024", "-depth", "24", "-fg", "-localhost", "no", "-SecurityTypes", "VncAuth", "-passwd", "/home/anotherme/.vnc/passwd"]