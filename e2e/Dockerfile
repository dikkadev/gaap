FROM ubuntu:latest

# Install required packages
RUN apt-get update && \
    apt-get install -y \
    curl \
    wget \
    git \
    sqlite3 \
    lsof \
    tree \
    && rm -rf /var/lib/apt/lists/*

# Create test user and set up environment
RUN useradd -m -s /bin/bash testuser && \
    mkdir -p /home/testuser/gaap/bin/actual /home/testuser/gaap/bin/links /home/testuser/gaap/db /home/testuser/gaap/config /home/testuser/gaap/logs && \
    chown -R testuser:testuser /home/testuser/gaap

# Copy the GAAP binary
COPY gaap /usr/local/bin/gaap
RUN chmod +x /usr/local/bin/gaap

# Switch to test user
USER testuser
WORKDIR /home/testuser

# Keep container running
CMD ["tail", "-f", "/dev/null"] 