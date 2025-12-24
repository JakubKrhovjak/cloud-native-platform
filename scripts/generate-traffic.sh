#!/bin/bash

# Traffic generator for student-service
# Sends messages to generate metrics
# Usage: ./scripts/generate-traffic.sh [OPTIONS]
#
# Options:
#   -n, --count NUM       Number of messages to send (default: 200)
#   -u, --url URL         Base URL (default: http://localhost:30080)
#   -e, --email EMAIL     Student email (default: test.student@example.com)
#   -p, --password PASS   Password (default: TestPassword123!)
#   -d, --delay SECONDS   Delay between requests in seconds (default: 0.01)
#   -h, --help           Show this help message

set -e

# Default values
MESSAGE_COUNT=200
BASE_URL="http://localhost:8080"  # Kind cluster maps NodePort 30080 to localhost:8080
EMAIL="test.student@example.com"
PASSWORD="TestPassword123!"
DELAY=0.01
COOKIE_FILE="/tmp/student_cookies_$(date +%s).txt"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--count)
      MESSAGE_COUNT="$2"
      shift 2
      ;;
    -u|--url)
      BASE_URL="$2"
      shift 2
      ;;
    -e|--email)
      EMAIL="$2"
      shift 2
      ;;
    -p|--password)
      PASSWORD="$2"
      shift 2
      ;;
    -d|--delay)
      DELAY="$2"
      shift 2
      ;;
    -h|--help)
      head -n 13 "$0" | grep '^#' | grep -v '#!/bin/bash' | sed 's/^# //'
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use -h or --help for usage information"
      exit 1
      ;;
  esac
done

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Traffic Generator for Student Service${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Endpoint:       ${YELLOW}$BASE_URL${NC}"
echo -e "Email:          ${YELLOW}$EMAIL${NC}"
echo -e "Message count:  ${YELLOW}$MESSAGE_COUNT${NC}"
echo -e "Delay:          ${YELLOW}${DELAY}s${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if service is reachable
echo -n "Checking service availability... "
if curl -s --connect-timeout 3 "$BASE_URL/health" > /dev/null 2>&1; then
  echo -e "${GREEN}✓${NC}"
else
  echo -e "${YELLOW}⚠${NC}"
  echo -e "${YELLOW}⚠ Cannot reach health endpoint at $BASE_URL/health${NC}"
  echo -e "${YELLOW}  Proceeding anyway, but check if service is running:${NC}"
  echo "  kubectl get svc -n grud student-service"
  echo "  kubectl get pods -n grud | grep student"
  echo ""
fi

# Try to login
echo -e "\n${BLUE}🔐 Logging in...${NC}"
LOGIN_RESPONSE=$(curl -s -c "$COOKIE_FILE" -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

# Check if login was successful
if ! echo "$LOGIN_RESPONSE" | grep -q "accessToken"; then
  echo -e "${YELLOW}📝 User doesn't exist or login failed, attempting to register...${NC}"

  REGISTER_RESPONSE=$(curl -s -c "$COOKIE_FILE" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"firstName\":\"Test\",\"lastName\":\"Student\",\"major\":\"Computer Science\",\"year\":1}")

  if ! echo "$REGISTER_RESPONSE" | grep -q "accessToken"; then
    echo -e "${RED}❌ Registration failed${NC}"
    echo "$REGISTER_RESPONSE"
    rm -f "$COOKIE_FILE"
    exit 1
  fi

  echo -e "${GREEN}✓${NC} User registered successfully"
else
  echo -e "${GREEN}✓${NC} Logged in successfully"
fi

# Get student ID from login/register response
STUDENT_ID=$(echo "$REGISTER_RESPONSE$LOGIN_RESPONSE" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
if [ -z "$STUDENT_ID" ]; then
  echo -e "${YELLOW}⚠ Could not extract student ID, skipping student view calls${NC}"
fi

# Send messages
echo -e "\n${BLUE}📨 Sending $MESSAGE_COUNT messages...${NC}"
FAILED=0
SUCCESS=0
STUDENT_VIEWS=0
STUDENTS_LIST_VIEWS=0
PROJECTS_LIST_VIEWS=0
START_TIME=$(date +%s)

for i in $(seq 1 $MESSAGE_COUNT); do
  RESPONSE=$(curl -s -b "$COOKIE_FILE" -X POST "$BASE_URL/api/messages" \
    -H "Content-Type: application/json" \
    -d "{\"message\":\"Load test message #$i - $(date +%H:%M:%S)\"}")

  # Check if response is successful (not an error)
  if echo "$RESPONSE" | grep -q "error\|invalid\|unauthorized"; then
    FAILED=$((FAILED + 1))
  else
    SUCCESS=$((SUCCESS + 1))
  fi

  # Every 10 messages, also call GetStudent API to generate view metrics
  if [ $((i % 10)) -eq 0 ] && [ -n "$STUDENT_ID" ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/students/$STUDENT_ID" > /dev/null
    STUDENT_VIEWS=$((STUDENT_VIEWS + 1))
  fi

  # Every 15 messages, call GetAllStudents API to generate list view metrics
  if [ $((i % 15)) -eq 0 ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/students" > /dev/null
    STUDENTS_LIST_VIEWS=$((STUDENTS_LIST_VIEWS + 1))
  fi

  # Every 20 messages, call GetAllProjects API to generate projects list view metrics
  if [ $((i % 20)) -eq 0 ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/projects" > /dev/null
    PROJECTS_LIST_VIEWS=$((PROJECTS_LIST_VIEWS + 1))
  fi

  # Progress indicator
  if [ $((i % 50)) -eq 0 ]; then
    PERCENT=$((i * 100 / MESSAGE_COUNT))
    echo -e "  ${YELLOW}[$PERCENT%]${NC} Sent $i/$MESSAGE_COUNT messages (Success: $SUCCESS, Failed: $FAILED)"
    echo -e "  ${BLUE}Stats:${NC} Student views: $STUDENT_VIEWS, Students list: $STUDENTS_LIST_VIEWS, Projects list: $PROJECTS_LIST_VIEWS"
  fi

  # Delay between requests
  if [ "$DELAY" != "0" ]; then
    sleep "$DELAY"
  fi
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Cleanup
rm -f "$COOKIE_FILE"

# Final summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ Traffic generation complete!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Messages sent:        ${GREEN}$MESSAGE_COUNT${NC}"
echo -e "Successful:           ${GREEN}$SUCCESS${NC}"
echo -e "Failed:               ${RED}$FAILED${NC}"
echo -e "Student views:        ${BLUE}$STUDENT_VIEWS${NC}"
echo -e "Students list views:  ${BLUE}$STUDENTS_LIST_VIEWS${NC}"
echo -e "Projects list views:  ${BLUE}$PROJECTS_LIST_VIEWS${NC}"
echo -e "Duration:             ${YELLOW}${DURATION}s${NC}"
if [ $DURATION -gt 0 ]; then
  RPS=$((MESSAGE_COUNT / DURATION))
  echo -e "Rate:           ${YELLOW}~${RPS} req/s${NC}"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📊 View metrics in Grafana:${NC}"
echo -e "   ${YELLOW}http://localhost:30300${NC} (admin/admin)"
echo ""
