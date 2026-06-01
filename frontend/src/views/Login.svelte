<div class="login">
  <div class="bird" aria-hidden="true">🐦‍🔥</div>
  <div class="content">
    <h1>Lifer</h1>
    <p class="tagline">Learn bird songs by ear</p>
    <p class="detail">Bird song identification practice</p>
    <a href="/api/v1/auth/google" class="google-btn">
      <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24">
        <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
        <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
        <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"/>
        <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
      </svg>
      Sign in with Google
    </a>
  </div>
</div>

<style>
  .login {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #0c2340 0%, #0f172a 55%, #1a0a0a 100%);
  }
  .bird {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16rem;
    opacity: 0.15;
    pointer-events: none;
    user-select: none;
  }
  .content {
    position: relative;
    text-align: center;
    padding: 2rem;
  }
  h1 {
    font-size: 2rem;
    font-weight: 700;
    color: #f1f5f9;
    letter-spacing: -0.03em;
    margin-bottom: 0.25rem;
  }
  .tagline {
    color: #94a3b8;
    font-size: 0.875rem;
    margin-bottom: 0.25rem;
  }
  .detail {
    color: #475569;
    font-size: 0.75rem;
    font-style: italic;
    margin-bottom: 1.5rem;
  }
  .google-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    background: #ffffff;
    color: #1c1917;
    border-radius: 8px;
    padding: 0.688rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    text-decoration: none;
    font-family: inherit;
  }
</style>
