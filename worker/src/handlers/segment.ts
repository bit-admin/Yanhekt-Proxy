import { Crypto } from '../crypto';
import { fetchVideoToken } from '../token';
import { fetchTSWithRetry } from '../fetch';
import {
  isAllowedVideoURL,
  isValidLoginToken,
  invalidVideoURLResponse,
  invalidTokenResponse,
} from '../validation';

interface SegmentHandlerOptions {
  request: Request;
  crypto: Crypto;
  upstreamAPI: string;
  magicKey: string;
  videoHost: string;
  tsFileName: string;
}

/**
 * Handles /ts/* endpoint
 */
export async function handleSegment(
  options: SegmentHandlerOptions
): Promise<Response> {
  const { request, crypto, upstreamAPI, magicKey, tsFileName } = options;

  // Handle CORS preflight
  if (request.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type',
      },
    });
  }

  const url = new URL(request.url);
  const baseURL = url.searchParams.get('base');
  const loginToken = url.searchParams.get('token');

  // Validate login token (missing or invalid format returns 403)
  if (!loginToken || !isValidLoginToken(loginToken)) {
    return invalidTokenResponse();
  }

  if (!baseURL) {
    return new Response('Missing required parameter: base', {
      status: 400,
      headers: { 'Access-Control-Allow-Origin': '*' },
    });
  }

  // Validate base URL domain
  if (!isAllowedVideoURL(baseURL)) {
    return invalidVideoURLResponse();
  }

  // URL decode the filename
  const decodedTsFileName = decodeURIComponent(tsFileName);

  // Build full TS URL
  const tsURL = resolveURL(baseURL, decodedTsFileName);

  // Validate resolved TS URL domain (in case of absolute URL in filename)
  if (!isAllowedVideoURL(tsURL)) {
    return invalidVideoURLResponse();
  }

  // Get fresh video token
  let videoToken: string;
  try {
    videoToken = await fetchVideoToken(loginToken, upstreamAPI, magicKey);
  } catch (error) {
    console.error('Failed to get video token:', error);
    return new Response('Failed to get video token', {
      status: 500,
      headers: { 'Access-Control-Allow-Origin': '*' },
    });
  }

  // Build signed URL function (for retry with fresh signature)
  const buildSignedURL = (): string => {
    const encryptedURL = crypto.encryptURL(tsURL);
    return crypto.signURL(encryptedURL, videoToken);
  };

  // Fetch TS with retry logic
  try {
    const response = await fetchTSWithRetry({
      getURL: buildSignedURL,
      videoHost: options.videoHost,
      onRetry: async () => {
        console.log('TS request retry, refreshing token');
        videoToken = await fetchVideoToken(loginToken, upstreamAPI, magicKey);
      },
    });

    // Create new response with CORS headers
    const headers = new Headers(response.headers);
    headers.set('Access-Control-Allow-Origin', '*');
    headers.set('Access-Control-Allow-Methods', 'GET, OPTIONS');
    headers.set('Access-Control-Allow-Headers', 'Content-Type');

    return new Response(response.body, {
      status: response.status,
      headers,
    });
  } catch (error) {
    console.error('Failed to fetch TS:', error);
    return new Response('Failed to fetch TS segment', {
      status: 502,
      headers: { 'Access-Control-Allow-Origin': '*' },
    });
  }
}

/**
 * Resolves a relative URL against a base URL
 */
function resolveURL(base: string, relative: string): string {
  if (relative.startsWith('http')) {
    return relative;
  }

  let baseURL: URL;
  try {
    baseURL = new URL(base);
  } catch {
    return relative;
  }

  if (relative.startsWith('/')) {
    return `${baseURL.protocol}//${baseURL.host}${relative}`;
  }

  // Relative path - append to base directory
  const basePath = baseURL.pathname;
  const lastSlash = basePath.lastIndexOf('/');
  const baseDir = lastSlash >= 0 ? basePath.substring(0, lastSlash + 1) : '/';

  return `${baseURL.protocol}//${baseURL.host}${baseDir}${relative}`;
}
