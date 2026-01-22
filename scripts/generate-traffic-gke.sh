#!/bin/bash

# Traffic generator for student-service on GKE
# Sends messages to generate metrics
# Usage: ./scripts/generate-traffic-gke.sh [OPTIONS]
#
# Options:
#   -n, --count NUM       Number of messages to send (default: 200)
#   -u, --url URL         Base URL (default: auto-detect from ingress)
#   -e, --email EMAIL     Student email (default: test.student@example.com)
#   -p, --password PASS   Password (default: TestPassword123!)
#   -d, --delay SECONDS   Delay between requests in seconds (default: 0.01)
#   -h, --help           Show this help message

set -e

# Default values
MESSAGE_COUNT=200
EMAIL="test.student@example.com"
PASSWORD="TestPassword123!"
DELAY=0.01
COOKIE_FILE="/tmp/student_cookies_$(date +%s).txt"

# Auto-detect GKE Gateway or Ingress URL
get_ingress_url() {
  # Try Gateway API first
  HOST=$(kubectl get gateway -n apps grud-gateway -o jsonpath='{.spec.listeners[0].hostname}' 2>/dev/null)
  if [ -n "$HOST" ] && [ "$HOST" != "*" ]; then
    echo "https://${HOST}"
    return
  fi
  # Try Gateway IP
  IP=$(kubectl get gateway -n apps grud-gateway -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)
  if [ -n "$IP" ]; then
    echo "https://grudapp.com"  # Use domain for proper cert
    return
  fi
  # Fallback to Ingress (legacy)
  HOST=$(kubectl get ingress -n apps grud-ingress -o jsonpath='{.spec.rules[0].host}' 2>/dev/null)
  if [ -n "$HOST" ]; then
    echo "https://${HOST}"
    return
  fi
  IP=$(kubectl get ingress -n apps grud-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
  if [ -n "$IP" ]; then
    echo "http://${IP}"
    return
  fi
}

# Try to get BASE_URL from ingress if not provided
DETECTED_URL=$(get_ingress_url)
if [ -n "$DETECTED_URL" ]; then
  BASE_URL="$DETECTED_URL"
else
  BASE_URL="http://localhost:8080"
  echo "⚠ Could not detect ingress, using localhost"
fi

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

echo -e "${BLUE}🚀 Traffic Generator for Student Service (GKE)${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "Endpoint:       ${YELLOW}$BASE_URL${NC}"
echo -e "Email:          ${YELLOW}$EMAIL${NC}"
echo -e "Message count:  ${YELLOW}$MESSAGE_COUNT${NC}"
echo -e "Delay:          ${YELLOW}${DELAY}s${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if service is reachable
echo -n "Checking service availability... "
HEALTH_RESPONSE=$(curl -s --connect-timeout 5 -w "%{http_code}" "$BASE_URL/health" -o /tmp/health_response.txt 2>&1)
if [ "$HEALTH_RESPONSE" = "200" ]; then
  echo -e "${GREEN}✓${NC}"
else
  echo -e "${YELLOW}⚠ HTTP $HEALTH_RESPONSE${NC}"
  echo -e "${YELLOW}  Response: $(cat /tmp/health_response.txt 2>/dev/null | head -c 100)${NC}"
  echo -e "${YELLOW}  Waiting for backend to become healthy...${NC}"
  echo ""
  echo "  Check backend health:"
  echo "  kubectl describe ingress -n apps grud-ingress | grep -A2 backends"
  echo ""
  read -p "Press Enter to continue anyway, or Ctrl+C to abort..."
fi

# Try to login
echo -e "\n${BLUE}🔐 Logging in...${NC}"
LOGIN_RESPONSE=$(curl -s -c "$COOKIE_FILE" -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" 2>&1)

# Check if login was successful
if ! echo "$LOGIN_RESPONSE" | grep -q "accessToken"; then
  echo -e "${YELLOW}📝 User doesn't exist or login failed, attempting to register...${NC}"

  REGISTER_RESPONSE=$(curl -s -c "$COOKIE_FILE" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"firstName\":\"Test\",\"lastName\":\"Student\",\"major\":\"Computer Science\",\"year\":1}" 2>&1)

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
    -d "{\"message\":\"Load test message #$i - $(date +%H:%M:%S)\"}" 2>&1)

  # Check if response is successful (not an error)
  if echo "$RESPONSE" | grep -q "error\|invalid\|unauthorized"; then
    FAILED=$((FAILED + 1))
  else
    SUCCESS=$((SUCCESS + 1))
  fi

  # Every 10 messages, also call GetStudent API to generate view metrics
  if [ $((i % 10)) -eq 0 ] && [ -n "$STUDENT_ID" ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/students/$STUDENT_ID" > /dev/null 2>&1
    STUDENT_VIEWS=$((STUDENT_VIEWS + 1))
  fi

  # Every 15 messages, call GetAllStudents API to generate list view metrics
  if [ $((i % 15)) -eq 0 ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/students" > /dev/null 2>&1
    STUDENTS_LIST_VIEWS=$((STUDENTS_LIST_VIEWS + 1))
  fi

  # Every 20 messages, call GetAllProjects API to generate projects list view metrics
  if [ $((i % 20)) -eq 0 ]; then
    curl -s -b "$COOKIE_FILE" "$BASE_URL/api/projects" > /dev/null 2>&1
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

# Get Grafana IP
GRAFANA_IP=$(kubectl get ingress -n infra grafana-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

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
  echo -e "Rate:                 ${YELLOW}~${RPS} req/s${NC}"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📊 View metrics in Grafana:${NC}"
if [ -n "$GRAFANA_IP" ]; then
  echo -e "   ${YELLOW}http://${GRAFANA_IP}${NC} (admin/admin)"
else
  echo -e "   ${YELLOW}kubectl port-forward -n infra svc/prometheus-grafana 3000:80${NC}"
  echo -e "   ${YELLOW}http://localhost:3000${NC} (admin/admin)"
fi
echo ""
