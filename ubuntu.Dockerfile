FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# https://mirrors.tuna.tsinghua.edu.cn/help/ubuntu/
RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list
RUN apt-get update && apt-get install -y \
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
    vim \
 && apt-get clean && rm -rf /var/lib/apt/lists/* \
 && update-alternatives --install /usr/bin/x-www-browser x-www-browser /usr/bin/chromium 50 \
 && update-alternatives --set x-www-browser /usr/bin/chromium

RUN useradd -m -s /bin/bash anotherme && \
    echo "anotherme:yourpassword" | chpasswd && \
    adduser anotherme sudo

USER anotherme
WORKDIR /home/anotherme
RUN mkdir -p .vnc && \
    echo "yourpassword" | vncpasswd -f > .vnc/passwd && \
    chmod 600 .vnc/passwd

RUN echo "#!/bin/bash\n\
unset SESSION_MANAGER\n\
unset DBUS_SESSION_BUS_ADDRESS\n\
export PULSE_SERVER=127.0.0.1\n\
startxfce4 &\n\
[ -x /etc/vnc/xstartup ] && exec /etc/vnc/xstartup\n\
[ -r \$HOME/.Xresources ] && xrdb \$HOME/.Xresources\n\
xsetroot -solid grey\n\
vncconfig -iconic &" > .vnc/xstartup && \
    chmod +x .vnc/xstartup

# 暴露 VNC 端口 (5900 + display number, 这里用 :1 即 5901)
EXPOSE 5901

CMD ["/usr/bin/vncserver", ":1", "-geometry", "1280x1024", "-depth", "24", "-fg", "-localhost", "no", "-SecurityTypes", "VncAuth", "-passwd", "/home/anotherme/.vnc/passwd"]