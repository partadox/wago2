#!/usr/bin/env python3
"""
Script to update all Send* methods in send.go to support multi-account
"""

import re

def update_send_go():
    file_path = '/Users/artakusuma/dev/wago2/src/usecase/send.go'

    with open(file_path, 'r') as f:
        content = f.read()

    # List of methods to update (excluding SendText which is already done)
    methods = [
        'SendImage',
        'SendFile',
        'SendVideo',
        'SendContact',
        'SendLink',
        'SendLocation',
        'SendAudio',
        'SendPoll',
        'SendPresence',
        'SendChatPresence',
        'SendSticker'
    ]

    for method in methods:
        # Pattern to find the method signature and first few lines
        pattern = rf'func \(service serviceSend\) {method}\(ctx context\.Context, request domainSend\.\w+Request\) \(response domainSend\.GenericResponse, err error\) {{\n(\s+)(err = validations\.)'

        # Replacement: add client retrieval before validation
        replacement = rf'''func (service serviceSend) {method}(ctx context.Context, request domainSend.\1Request) (response domainSend.GenericResponse, err error) {{
\1// Get client for account
\1client, err := service.getClient(request.BaseRequest.AccountID)
\1if err != nil {{
\1\1return response, err
\1}}

\1\2'''

        # Try to match and replace (for methods with BaseRequest)
        if re.search(pattern, content):
            content = re.sub(pattern, replacement, content, count=1)
            print(f"Updated {method}")

    # Special handling for SendPresence (no BaseRequest.Phone)
    content = re.sub(
        r'func \(service serviceSend\) SendPresence\(ctx context\.Context, request domainSend\.PresenceRequest\) \(response domainSend\.GenericResponse, err error\) \{\n\terr = validations\.',
        '''func (service serviceSend) SendPresence(ctx context.Context, request domainSend.PresenceRequest) (response domainSend.GenericResponse, err error) {
\t// Get client for account
\tclient, err := service.getClient(request.AccountID)
\tif err != nil {
\t\treturn response, err
\t}

\terr = validations.''',
        content
    )

    # Replace all utils.ValidateJidWithLogin(whatsapp.GetClient(), ...) with client
    content = re.sub(
        r'utils\.ValidateJidWithLogin\(whatsapp\.GetClient\(\), request\.(?:BaseRequest\.)?Phone\)',
        'utils.ValidateJidWithLogin(client, request.BaseRequest.Phone)',
        content
    )
    content = re.sub(
        r'utils\.ValidateJidWithLogin\(whatsapp\.GetClient\(\), request\.Phone\)',
        'utils.ValidateJidWithLogin(client, request.Phone)',
        content
    )

    # Replace uploadMedia calls
    content = re.sub(
        r'service\.uploadMedia\(ctx, whatsmeow',
        'service.uploadMedia(ctx, client, whatsmeow',
        content
    )

    # Replace wrapSendMessage calls
    content = re.sub(
        r'service\.wrapSendMessage\(ctx, dataWaRecipient',
        'service.wrapSendMessage(ctx, client, dataWaRecipient',
        content
    )
    content = re.sub(
        r'service\.wrapSendMessage\(ctx, userJid',
        'service.wrapSendMessage(ctx, client, userJid',
        content
    )

    # Replace client.BuildPollCreation
    content = re.sub(
        r'whatsapp\.GetClient\(\)\.BuildPollCreation',
        'client.BuildPollCreation',
        content
    )

    # Replace SendPresence calls
    content = re.sub(
        r'whatsapp\.GetClient\(\)\.SendPresence\(ctx',
        'client.SendPresence(ctx',
        content
    )

    # Replace SendChatPresence calls
    content = re.sub(
        r'whatsapp\.GetClient\(\)\.SendChatPresence\(ctx',
        'client.SendChatPresence(ctx',
        content
    )

    # Update getMentionFromText to accept client parameter
    content = re.sub(
        r'func \(service serviceSend\) getMentionFromText\(_ context\.Context, messages string\) \(result \[\]string\) \{',
        'func (service serviceSend) getMentionFromText(_ context.Context, client *whatsmeow.Client, messages string) (result []string) {',
        content
    )

    # Update calls to getMentionFromText
    content = re.sub(
        r'service\.getMentionFromText\(ctx, request\.Message\)',
        'service.getMentionFromText(ctx, client, request.Message)',
        content
    )

    # Update the getMentionFromText body
    content = re.sub(
        r'if dataWaRecipient, err := utils\.ValidateJidWithLogin\(whatsapp\.GetClient\(\), mention\); err == nil \{',
        'if dataWaRecipient, err := utils.ValidateJidWithLogin(client, mention); err == nil {',
        content
    )

    # Write back
    with open(file_path, 'w') as f:
        f.write(content)

    print("\n✅ All Send* methods updated!")
    print("Please run 'go build' to check for any remaining errors")

if __name__ == '__main__':
    update_send_go()
