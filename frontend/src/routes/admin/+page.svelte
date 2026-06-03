<div class="admin-home">
  <h1>Admin</h1>
  <ul class="section-list">
    <li><a href="/admin/species">Species</a></li>
  </ul>
</div>

<style>
  .admin-home {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h1 {
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }
  .section-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .section-list li {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    box-shadow: var(--shadow);
  }
  .section-list a {
    display: block;
    padding: 0.875rem 1rem;
    color: var(--text);
    text-decoration: none;
    font-weight: 600;
    font-size: 0.9375rem;
  }
  .section-list a:hover {
    color: var(--accent);
  }
</style>
