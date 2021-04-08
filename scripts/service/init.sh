sudo cp chainmaker.service  /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl start chainmaker
sudo systemctl enable chainmaker
sudo systemctl status chainmaker
