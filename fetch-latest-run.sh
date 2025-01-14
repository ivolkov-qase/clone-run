#!/bin/bash

# Ensure that the required environment variables are set
if [ -z "$QASE_PROJECT_CODE" ] || [ -z "$QASE_API_TOKEN" ]; then
  echo "Error: QASE_PROJECT_CODE and QASE_API_TOKEN must be set as environment variables."
  exit 1
fi

# Check if running on macOS or Linux and set the date command accordingly
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS: Use `-v` to subtract 1 month
  START_TIME=$(date -v-1m +%s)
else
  # Linux: Use `-d` to subtract 1 month
  START_TIME=$(date -d "1 month ago" +%s)
fi

# API URL base
API_URL="https://api.qase.io/v1/run/$QASE_PROJECT_CODE?from_start_time=$START_TIME&limit=100&offset="

# Initialize variables
OFFSET=0
LATEST_RUN_ID=0

# Loop to handle pagination
while true; do
  # Make the API request using curl
  RESPONSE=$(curl --silent --request GET \
    --url "${API_URL}${OFFSET}" \
    --header "Token: $QASE_API_TOKEN" \
    --header "accept: application/json")
  
  # Conditionally print the API response if QASE_DEBUG is set to true
  if [ "$QASE_DEBUG" == "true" ]; then
    echo "API Response: $RESPONSE"
  fi

  # Check if the API response contains any error
  if echo "$RESPONSE" | grep -q '"error"'; then
    echo "Error: API request failed. Response:"
    echo "$RESPONSE"
    exit 1
  fi

  # Get the total number of runs (to determine if we need more requests)
  TOTAL=$(echo "$RESPONSE" | jq '.result.total // 0')
  
  # If no runs exist, break the loop
  if [ "$TOTAL" -eq 0 ]; then
    echo "No runs found."
    break
  fi

  # Get the run IDs from the response
  RUN_IDS=$(echo "$RESPONSE" | jq -r '.result.entities[].id')

  # Check if no run IDs are found
  if [ -z "$RUN_IDS" ]; then
    echo "No run IDs found in the response."
    break
  fi

  # Loop through each run ID and keep track of the latest one
  for ID in $RUN_IDS; do
    if [ "$ID" -gt "$LATEST_RUN_ID" ]; then
      LATEST_RUN_ID=$ID
    fi
  done

  # Increment the OFFSET to fetch the next page
  OFFSET=$((OFFSET + 100))

  # If we've fetched all the runs, break the loop
  if [ "$OFFSET" -ge "$TOTAL" ]; then
    break
  fi
done

# Display the latest run ID
echo "The latest run ID is: $LATEST_RUN_ID"
