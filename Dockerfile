# 1. 使用轻量级的基础镜像
FROM ubuntu:20.04

# 2. 创建 /home/dcgm 目录并设置为工作目录
RUN mkdir -p /home/dcgm/lib
WORKDIR /home/dcgm

# 3. 更新包列表并安装 kmod
RUN apt update && apt install -y kmod pciutils

# 4. 复制已编译好的二进制文件到 /home/dcgm 目录
COPY dcgm-dcu /home/dcgm/dcgm-dcu

# 5. 复制 .so 依赖库到 /home/dcgm/lib 目录
RUN mkdir -p /home/dcgm/lib/driver6.2.x /home/dcgm/lib/driver6.3.x

# 5. 复制库文件
COPY pkg/dcgm/lib/driver6.2.x/librocm_smi64.so.2.8 /home/dcgm/lib/driver6.2.x/
COPY pkg/dcgm/lib/driver6.2.x/libhydmi.so.1.5 /home/dcgm/lib/driver6.2.x/
COPY pkg/dcgm/lib/driver6.2.x/libhydmi_mig.so.1.3 /home/dcgm/lib/driver6.2.x/
COPY pkg/dcgm/lib/driver6.3.x/librocm_smi64.so.2.8 /home/dcgm/lib/driver6.3.x/
COPY pkg/dcgm/lib/driver6.3.x/libhydmi.so.1.5 /home/dcgm/lib/driver6.3.x/
COPY pkg/dcgm/lib/driver6.3.x/libhydmi_mig.so.1.3 /home/dcgm/lib/driver6.3.x/

# 6. 设置库文件权限（使用简单的 chmod 命令）
RUN chmod 755 /home/dcgm/lib/driver6.2.x/lib*.so.* && \
    chmod 755 /home/dcgm/lib/driver6.3.x/lib*.so.*

# 7. 创建符号链接
RUN cd /home/dcgm/lib/driver6.2.x && \
    ln -sf librocm_smi64.so.2.8 librocm_smi64.so.2 && \
    ln -sf librocm_smi64.so.2 librocm_smi64.so && \
    ln -sf libhydmi.so.1.5 libhydmi.so.1 && \
    ln -sf libhydmi.so.1 libhydmi.so && \
    ln -sf libhydmi_mig.so.1.3 libhydmi_mig.so.1 && \
    ln -sf libhydmi_mig.so.1 libhydmi_mig.so && \
    cd /home/dcgm/lib/driver6.3.x && \
    ln -sf librocm_smi64.so.2.8 librocm_smi64.so.2 && \
    ln -sf librocm_smi64.so.2 librocm_smi64.so && \
    ln -sf libhydmi.so.1.5 libhydmi.so.1 && \
    ln -sf libhydmi.so.1 libhydmi.so && \
    ln -sf libhydmi_mig.so.1.3 libhydmi_mig.so.1 && \
    ln -sf libhydmi_mig.so.1 libhydmi_mig.so

# 8. 设置可执行权限
RUN chmod +x /home/dcgm/dcgm-dcu

# 9. 暴露端口
EXPOSE 16081

# 10. 启动命令
CMD ["/bin/sh", "-c", "/home/dcgm/dcgm-dcu -logtostderr -v=5 > /home/dcgm/dcgm.log 2>&1"]
