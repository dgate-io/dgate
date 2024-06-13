# install promtail 

sudo apt-get install -y unzip
wget https://github.com/grafana/loki/releases/download/v2.9.8/promtail-linux-amd64.zip
unzip promtail-linux-amd64.zip
chmod +x promtail-linux-amd64
sudo rm promtail-linux-amd64.zip
sudo mv promtail-linux-amd64 /usr/bin/promtail
sudo touch /etc/systemd/system/promtail.service

sudo bash -c 'cat <<EOF > /etc/systemd/system/promtail.service
[Unit]
Description=Promtail service
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/bin/promtail -config.file /etc/promtail/config.yml
TimeoutSec = 60
Restart = on-failure
RestartSec = 2

[Install]
WantedBy=multi-user.target

EOF'

sudo mkdir /etc/promtail
sudo touch /etc/promtail/config.yml
sudo bash -c 'cat <<EOF > /etc/promtail/config.yml
server:
  http_listen_port: 9088
  grpc_listen_port: 0
positions:
  filename: /tmp/positions.yaml
target_config:
  sync_period: 10s
clients:
- url: http://10.146.0.3:3100/loki/api/v1/push

scrape_configs:
  - job_name: journal
    journal:
      json: false
      max_age: 72h
      path: /var/log/journal
      labels:
        job: journal
        server: dgate-madrid
    relabel_configs:
      - source_labels: ['__journal__systemd_unit']
        target_label: 'unit'
      - source_labels: ['__journal__hostname']
        target_label: 'hostname'
      - source_labels: ['__journal__boot_id']
        target_label: 'boot_id'
      - source_labels: ['__journal__transport']
        target_label: 'transport'
      - source_labels: ['__journal__syslog_identifier']
        target_label: 'syslog_id'
      - source_labels: ["__journal_priority_keyword"]
        target_label: level

EOF'

sudo systemctl daemon-reload
sudo systemctl enable promtail
sudo systemctl start promtail
sudo systemctl status promtail







# install node_exporter
wget https://github.com/prometheus/node_exporter/releases/download/v1.8.1/node_exporter-1.8.1.linux-amd64.tar.gz
tar -xvf node_exporter-1.8.1.linux-amd64.tar.gz
sudo mv node_exporter-1.8.1.linux-amd64/node_exporter /usr/local/bin/
sudo touch /etc/systemd/system/node_exporter.service
sudo bash -c 'cat <<EOF > /etc/systemd/system/node_exporter.service
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=root
Group=root
Type=simple
Restart=on-failure
RestartSec=5s
ExecStart=/usr/local/bin/node_exporter --collector.logind --collector.processes --collector.systemd

[Install]
WantedBy=multi-user.target

EOF'

sudo systemctl daemon-reload
sudo systemctl enable node_exporter
sudo systemctl start node_exporter
sudo systemctl status node_exporter
