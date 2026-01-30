const ALLOWED_VIDEO_HOST = 'cvideo.yanhekt.cn';
const LOGIN_TOKEN_REGEX = /^[a-fA-F0-9]{32}$/;

/**
 * Validates that a URL points to the allowed video domain
 */
export function isAllowedVideoURL(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.host === ALLOWED_VIDEO_HOST;
  } catch {
    return false;
  }
}

/**
 * Validates that a login token is exactly 32 hex characters
 */
export function isValidLoginToken(token: string): boolean {
  return LOGIN_TOKEN_REGEX.test(token);
}

/**
 * Returns an error response for invalid video URL
 */
export function invalidVideoURLResponse(): Response {
  return new Response(
    `Invalid video URL: only ${ALLOWED_VIDEO_HOST} is allowed`,
    {
      status: 400,
      headers: { 'Access-Control-Allow-Origin': '*' },
    }
  );
}

/**
 * Returns an error response for invalid or missing login token
 */
export function invalidTokenResponse(): Response {
  return new Response('Forbidden', {
    status: 403,
    headers: { 'Access-Control-Allow-Origin': '*' },
  });
}
