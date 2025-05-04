FROM postgres:16-bookworm

ARG PGVECTOR_VERSION=v0.8.0
ARG AGE_BRANCH=master
# 接收来自 docker compose build args 的代理设置
ARG ALL_PROXY

# 切换到 root 安装依赖
USER root

ENV ALL_PROXY=${ALL_PROXY}
ENV HTTPS_PROXY=${ALL_PROXY}
RUN echo "Configuring APT sources to USTC mirror using .sources format..." \
    && sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list || true \
    && sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources || true \
    # pgdg apt.postgresql.org/pub/repos/apt -> mirrors.ustc.edu.cn/postgresql/repos/apt/
    && sed -i  's/apt.postgresql.org\/pub/mirrors.ustc.edu.cn\/postgresql/g' /etc/apt/sources.list.d/pgdg.list \
    && echo "APT sources configured."

# Install build dependencies for the extensions
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    git \
    wget \
    libreadline-dev \
    zlib1g-dev \
    flex \
    bison \
    libicu-dev \
    # Build dependencies for postgres itself (needed for extensions)
    libpq-dev \
    postgresql-server-dev-16 \
    tzdata \
    ca-certificates \
    locales \
    # Cleanup lists
    && rm -rf /var/lib/apt/lists/*

# --- Install pgvector ---
WORKDIR /tmp
RUN git clone --branch ${PGVECTOR_VERSION} https://github.com/pgvector/pgvector.git \
    && cd pgvector \
    && make \
    && make install

# --- Install scws (dependency for zhparser) ---
WORKDIR /tmp
RUN wget -q -O - http://www.xunsearch.com/scws/down/scws-1.2.3.tar.bz2 | tar xjf - \
    && cd scws-1.2.3 \
    && ./configure \
    && make install

# --- Install zhparser ---
WORKDIR /tmp
RUN git clone https://github.com/amutu/zhparser.git \
    && cd zhparser \
    && make \
    && make install

# --- Install Apache AGE ---
WORKDIR /tmp
RUN git clone https://github.com/apache/age.git \
    && cd age \
    && git checkout ${AGE_BRANCH} \
    && make PG_CONFIG=/usr/bin/pg_config install

# 设置 locale
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# --- 取消代理设置 ---
# 在完成所有需要代理的网络操作后，取消代理环境变量，避免泄露到最终镜像
ENV ALL_PROXY=""
ENV HTTPS_PROXY=""

# 设置时区
ENV TZ=Asia/Shanghai

# --- Cleanup ---
RUN apt-get purge -y --auto-remove build-essential cmake postgresql-server-dev-16 && apt-get clean
RUN rm -rf /tmp/*

# 配置 postgres 初始化脚本
COPY init-extensions.sh /docker-entrypoint-initdb.d/
RUN chmod +x /docker-entrypoint-initdb.d/init-extensions.sh

# 切换回 postgres 用户
USER postgres

# 重置工作目录
WORKDIR /

# 修改默认命令以预加载 AGE 库
# 基础镜像的 entrypoint 将执行此命令
CMD ["postgres", "-c", "shared_preload_libraries=age"]