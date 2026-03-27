./book-dragon-api &
SERVER_PID=$!
sleep 1

echo "--- Register ---"
curl -s -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{"username": "test2", "email": "test2@example.com", "password": "password123"}'
echo ""

echo "--- Login ---"
curl -s -X POST http://localhost:8080/login -H "Content-Type: application/json" -d '{"email": "test2@example.com", "password": "password123"}' > login.json
cat login.json
echo ""

TOKEN=$(grep -o '"token":"[^"]*' login.json | grep -o '[^"]*$')
echo "--- Auth/Me ---"
curl -s -X GET http://localhost:8080/auth/me -H "Authorization: Bearer $TOKEN"
echo ""

kill $SERVER_PID
