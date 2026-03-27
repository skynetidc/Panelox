$env:SSHPASS = "QsUUqVFStYYshort"
$SERVER = "root@89.203.249.68"
$FILE = "C:\Users\jakub\projects\panelox\panelox-release.tar.gz"

# Accept host key
Write-Host "Accepting host key..."
echo "y" | ssh -o StrictHostKeyChecking=no $SERVER "echo ok" 2>$null

# Upload
Write-Host "Uploading panelox-release.tar.gz..."
scp -o StrictHostKeyChecking=no $FILE "${SERVER}:/tmp/"

# Install
Write-Host "Installing on server..."
ssh -o StrictHostKeyChecking=no $SERVER @'
set -e
echo "=== Installing dependencies ==="
apt update -qq && apt install -y -qq docker.io postgresql postgresql-client nginx curl > /dev/null 2>&1
systemctl enable --now docker postgresql

echo "=== Setting up PostgreSQL ==="
sudo -u postgres psql -c "CREATE USER panelox WITH PASSWORD 'devpassword';" 2>/dev/null || true
sudo -u postgres psql -c "CREATE DATABASE panelox OWNER panelox;" 2>/dev/null || true

echo "=== Deploying Panelox ==="
mkdir -p /opt/panelox
cd /opt/panelox
tar -xzf /tmp/panelox-release.tar.gz
chmod +x panelox
cp .env.example .env

echo "=== Starting Panelox ==="
nohup ./panelox > /var/log/panelox.log 2>&1 &
sleep 3

echo "=== Seeding data ==="
bash scripts/seed.sh http://localhost:8080/api/v1

echo "=== Testing API ==="
bash scripts/test-api.sh http://localhost:8080/api/v1

echo ""
echo "=== DONE ==="
echo "Panel: http://89.203.249.68:8080"
echo "Admin: admin@panelox.io / admin123"
'@
