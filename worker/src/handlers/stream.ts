import { Crypto } from '../crypto';
import { fetchVideoToken } from '../token';
import { fetchM3U8WithRetry } from '../fetch';
import {
  isAllowedVideoURL,
  isValidLoginToken,
  invalidVideoURLResponse,
  invalidTokenResponse,
} from '../validation';

interface StreamHandlerOptions {
  request: Request;
  crypto: Crypto;
  upstreamAPI: string;
  magicKey: string;
  videoHost: string;
}

/**
 * Handles /stream endpoint
 */
export async function handleStream(
  options: StreamHandlerOptions
): Promise<Response> {
  const { request, crypto, upstreamAPI, magicKey } = options;

  const url = new URL(request.url);
  const originalURL = url.searchParams.get('url');
  const loginToken = url.searchParams.get('token');

  // Validate login token (missing or invalid format returns 403)
  if (!loginToken || !isValidLoginToken(loginToken)) {
    return invalidTokenResponse();
  }

  if (!originalURL) {
    return new Response('Missing required parameter: url', {
      status: 400,
      headers: { 'Access-Control-Allow-Origin': '*' },
    });
  }

  // Fix URL escaping
  const fixedURL = originalURL.replace(/\\\//g, '/');

  // Validate video URL domain
  if (!isAllowedVideoURL(fixedURL)) {
    return invalidVideoURLResponse();
  }

  // Get fresh video token
  let videoToken: string;
  try {
    videoToken = await fetchVideoToken(loginToken, upstreamAPI, magicKey);
  } catch (error) {
    console.error('Failed to get video token:', error);
    return new Response('Failed to get video token', { status: 500 });
  }

  // Build signed URL function (for retry with fresh signature)
  const buildSignedURL = (): string => {
    const encryptedURL = crypto.encryptURL(fixedURL);
    return crypto.signURL(encryptedURL, videoToken);
  };

  // Fetch M3U8 with retry logic
  let content: string;
  try {
    content = await fetchM3U8WithRetry({
      getURL: buildSignedURL,
      videoHost: options.videoHost,
      onRetry: async () => {
        console.log('M3U8 request retry, refreshing token');
        videoToken = await fetchVideoToken(loginToken, upstreamAPI, magicKey);
      },
    });
  } catch (error) {
    console.error('Failed to fetch M3U8:', error);
    return new Response('Failed to fetch M3U8', { status: 502 });
  }

  // Rewrite TS URLs in M3U8 content
  const rewrittenContent = rewriteM3U8Content(
    content,
    fixedURL,
    loginToken,
    request
  );

  return new Response(rewrittenContent, {
    status: 200,
    headers: {
      'Content-Type': 'application/vnd.apple.mpegurl',
      'Access-Control-Allow-Origin': '*',
    },
  });
}

/**
 * Rewrites TS segment URLs
 */
function rewriteM3U8Content(
  content: string,
  baseURL: string,
  loginToken: string,
  request: Request
): string {
  const lines = content.split('\n');
  const result: string[] = [];

  const requestURL = new URL(request.url);
  const serverHost = requestURL.host;
  const scheme = requestURL.protocol.replace(':', '');
  const pathPrefix = '/ts/';

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed === '' || trimmed.startsWith('#')) {
      result.push(line);
      continue;
    }

    const tsFileName = trimmed;
    const rewrittenURL = `${scheme}://${serverHost}${pathPrefix}${encodeURIComponent(tsFileName)}?base=${encodeURIComponent(baseURL)}&token=${encodeURIComponent(loginToken)}`;
    result.push(rewrittenURL);
  }

  return result.join('\n');
}
