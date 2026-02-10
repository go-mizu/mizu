import { graphqlBaseURL, gqlFeatures, userAgent } from './config'
import { generateTID } from './tid'
import { X_AUTH_TOKEN, X_CT0, X_BEARER_TOKEN } from '../env'

export class GraphQLClient {
  private authToken: string
  private ct0: string
  private bearerToken: string

  constructor(authToken: string, ct0: string, bearerToken: string) {
    this.authToken = authToken
    this.ct0 = ct0
    this.bearerToken = bearerToken
  }

  async doGraphQL(
    endpoint: string,
    variables: Record<string, unknown>,
    fieldToggles?: string
  ): Promise<Record<string, unknown>> {
    const params = new URLSearchParams()
    params.set('variables', JSON.stringify(variables))
    params.set('features', gqlFeatures)
    if (fieldToggles) {
      params.set('fieldToggles', fieldToggles)
    }

    const fullURL = graphqlBaseURL + endpoint + '?' + params.toString()
    const url = new URL(fullURL)
    const apiPath = url.pathname

    let tid = ''
    try {
      tid = await generateTID(apiPath)
      console.log('[GQL] TID generated:', tid.length, 'chars for', endpoint.split('/').pop())
    } catch (e: any) {
      console.warn('[GQL] TID generation failed:', e.message)
    }

    const headers: Record<string, string> = {
      accept: '*/*',
      'accept-language': 'en-US,en;q=0.9',
      'content-type': 'application/json',
      origin: 'https://x.com',
      'user-agent': userAgent,
      'x-twitter-active-user': 'yes',
      'x-twitter-client-language': 'en',
      priority: 'u=1, i',
      authorization: this.bearerToken,
      'x-twitter-auth-type': 'OAuth2Session',
      'x-csrf-token': this.ct0,
      cookie: `auth_token=${this.authToken}; ct0=${this.ct0}`,
      'sec-ch-ua': '"Google Chrome";v="142", "Chromium";v="142", "Not A(Brand";v="24"',
      'sec-ch-ua-mobile': '?0',
      'sec-ch-ua-platform': '"Windows"',
      'sec-fetch-dest': 'empty',
      'sec-fetch-mode': 'cors',
      'sec-fetch-site': 'same-site',
    }

    if (tid) {
      headers['x-client-transaction-id'] = tid
    }

    console.log('[GQL] Fetching:', endpoint.split('/').pop(), 'tid:', tid ? 'yes' : 'NO')
    const resp = await fetch(fullURL, { headers })
    console.log('[GQL] Response:', resp.status, 'for', endpoint.split('/').pop())

    if (resp.status === 429) {
      throw new Error('rate limited (429)')
    }

    const body = await resp.text()

    if (resp.status !== 200) {
      const errBody = body.length > 200 ? body.slice(0, 200) : body
      throw new Error(`HTTP ${resp.status}: ${errBody}`)
    }

    const result = JSON.parse(body) as Record<string, unknown>

    const errors = result.errors as Array<Record<string, unknown>> | undefined
    if (errors && errors.length > 0) {
      const first = errors[0]
      const msg = (first.message as string) || ''
      const code = (first.code as number) || 0
      switch (code) {
        case 88:
          throw new Error(`rate limited: ${msg}`)
        case 89:
          throw new Error(`expired token: ${msg}`)
        case 239:
          throw new Error(`bad token: ${msg}`)
        case 326:
          throw new Error(`account locked: ${msg}`)
        case 37:
          throw new Error(`user suspended: ${msg}`)
        default:
          if (result.data) return result
          throw new Error(`API error ${code}: ${msg}`)
      }
    }

    return result
  }
}

// Prebuilt client from env credentials (mirrors x-viewer's env approach)
let _client: GraphQLClient | null = null
export function getClient(): GraphQLClient {
  if (_client) return _client
  if (!X_AUTH_TOKEN || !X_CT0 || !X_BEARER_TOKEN) {
    throw new Error('Credentials not configured. Copy env.example.ts to env.ts and fill in values.')
  }
  _client = new GraphQLClient(X_AUTH_TOKEN, X_CT0, X_BEARER_TOKEN)
  return _client
}

export function resetClient() {
  _client = null
}
