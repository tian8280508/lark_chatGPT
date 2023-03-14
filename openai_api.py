import sys

import openai
openai.api_key = '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'


def openAIReq():
    # print('send request to openai_api')
    res = openai.ChatCompletion.create(
        model="gpt-3.5-turbo",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": sys.argv[1]},
        ]
    )
    # print('receive response from openai_api')
    print(res.choices[0].message.content)

openAIReq()
