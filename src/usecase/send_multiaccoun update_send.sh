#!/bin/bash

# Script to update send.go untuk multi-account support
# Replaces whatsapp.GetClient() with proper client based on account_id

cd /Users/artakusuma/dev/wago2/src/usecase

# Backup original file
cp send.go send.go.backup

# Replace utils.ValidateJidWithLogin(whatsapp.GetClient(), ...) with utils.ValidateJidWithLogin(client, ...)
sed -i '' 's/utils\.ValidateJidWithLogin(whatsapp\.GetClient(), request\.BaseRequest\.Phone)/utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)/g' send.go
sed -i '' 's/utils\.ValidateJidWithLogin(whatsapp\.GetClient(), request\.Phone)/utils.ValidateJidWithLogin(client, request.Phone)/g' send.go

# Replace service.uploadMedia(ctx, whatsmeow... with service.uploadMedia(ctx, client, whatsmeow...
sed -i '' 's/service\.uploadMedia(ctx, whatsmeow/service.uploadMedia(ctx, client, whatsmeow/g' send.go

# Replace service.wrapSendMessage(ctx, with service.wrapSendMessage(ctx, client,
sed -i '' 's/service\.wrapSendMessage(ctx, /service.wrapSendMessage(ctx, client, /g' send.go

# Replace whatsapp.GetClient().BuildPollCreation with client.BuildPollCreation
sed -i '' 's/whatsapp\.GetClient()\.BuildPollCreation/client.BuildPollCreation/g' send.go

# Replace whatsapp.GetClient().SendPresence with client.SendPresence
sed -i '' 's/whatsapp\.GetClient()\.SendPresence/client.SendPresence/g' send.go

# Replace whatsapp.GetClient().SendChatPresence with client.SendChatPresence
sed -i '' 's/whatsapp\.GetClient()\.SendChatPresence/client.SendChatPresence/g' send.go

# In getMentionFromText, replace whatsapp.GetClient() with client (but we need to pass client as param)
# This is tricky, let's leave it for now and handle separately

echo "Script completed. Please check send.go for any remaining whatsapp.GetClient() calls"
echo "You'll need to manually add client retrieval at the start of each Send* method"
