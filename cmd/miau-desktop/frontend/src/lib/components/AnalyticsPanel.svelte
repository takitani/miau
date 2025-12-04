<script>
  import { onMount } from 'svelte';
  import { analyticsData, analyticsLoading, analyticsPeriod, loadAnalytics } from '../stores/analytics.js';

  // Period options
  const periods = [
    { value: '7d', label: '7 dias' },
    { value: '30d', label: '30 dias' },
    { value: '90d', label: '90 dias' },
    { value: 'all', label: 'Todos' }
  ];

  function selectPeriod(period) {
    loadAnalytics(period);
  }

  function formatNumber(num) {
    if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'k';
    }
    return num.toString();
  }

  function formatTime(minutes) {
    if (minutes < 60) {
      return Math.round(minutes) + ' min';
    }
    const hours = Math.floor(minutes / 60);
    const mins = Math.round(minutes % 60);
    if (hours < 24) {
      return hours + 'h ' + mins + 'm';
    }
    const days = Math.floor(hours / 24);
    return days + 'd ' + (hours % 24) + 'h';
  }

  function getBarWidth(count, max) {
    if (max === 0) return 0;
    return (count / max) * 100;
  }

  onMount(() => {
    if (!$analyticsData) {
      loadAnalytics($analyticsPeriod);
    }
  });

  // Calculate max for bar charts
  $: maxHourly = $analyticsData?.trends?.hourly?.reduce((max, h) => Math.max(max, h.count), 0) || 1;
  $: maxWeekday = $analyticsData?.trends?.weekday?.reduce((max, w) => Math.max(max, w.count), 0) || 1;
  $: maxSender = $analyticsData?.topSenders?.[0]?.count || 1;
</script>

<div class="analytics-panel">
  <!-- Period selector -->
  <div class="period-selector">
    {#each periods as period}
      <button
        class="period-btn"
        class:active={$analyticsPeriod === period.value}
        on:click={() => selectPeriod(period.value)}
      >
        {period.label}
      </button>
    {/each}
  </div>

  {#if $analyticsLoading}
    <div class="loading">
      <span class="spinner"></span>
      Carregando estatísticas...
    </div>
  {:else if $analyticsData}
    <!-- Overview Cards -->
    <div class="overview-cards">
      <div class="card">
        <div class="card-value">{formatNumber($analyticsData.overview.totalEmails)}</div>
        <div class="card-label">Total Emails</div>
      </div>
      <div class="card unread">
        <div class="card-value">{formatNumber($analyticsData.overview.unreadEmails)}</div>
        <div class="card-label">Não Lidos</div>
      </div>
      <div class="card">
        <div class="card-value">{formatNumber($analyticsData.overview.sentEmails)}</div>
        <div class="card-label">Enviados</div>
      </div>
      <div class="card">
        <div class="card-value">{$analyticsData.overview.storageUsedMb.toFixed(1)} MB</div>
        <div class="card-label">Armazenamento</div>
      </div>
    </div>

    <!-- Response Stats -->
    <div class="section">
      <h3>Tempo de Resposta</h3>
      <div class="response-stats">
        <div class="stat">
          <span class="stat-value">{formatTime($analyticsData.responseTime.avgResponseMinutes)}</span>
          <span class="stat-label">Média</span>
        </div>
        <div class="stat">
          <span class="stat-value">{$analyticsData.responseTime.responseRate.toFixed(1)}%</span>
          <span class="stat-label">Taxa de Resposta</span>
        </div>
      </div>
    </div>

    <!-- Top Senders -->
    <div class="section">
      <h3>Top Remetentes</h3>
      <div class="top-senders">
        {#each ($analyticsData.topSenders || []).slice(0, 5) as sender, i}
          <div class="sender-row">
            <div class="sender-rank">{i + 1}</div>
            <div class="sender-info">
              <div class="sender-name">{sender.name || sender.email}</div>
              <div class="sender-email">{sender.email}</div>
            </div>
            <div class="sender-stats">
              <div class="sender-bar-container">
                <div class="sender-bar" style="width: {getBarWidth(sender.count, maxSender)}%"></div>
              </div>
              <span class="sender-count">{sender.count}</span>
              {#if sender.unreadCount > 0}
                <span class="sender-unread">({sender.unreadCount} new)</span>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Hourly Distribution -->
    <div class="section">
      <h3>Distribuição por Hora</h3>
      <div class="hourly-chart">
        {#each ($analyticsData.trends?.hourly || []) as hour}
          <div class="hour-bar-container" title="{hour.hour}h: {hour.count} emails">
            <div
              class="hour-bar"
              style="height: {getBarWidth(hour.count, maxHourly)}%"
            ></div>
            {#if hour.hour % 6 === 0}
              <span class="hour-label">{hour.hour}h</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Weekday Distribution -->
    <div class="section">
      <h3>Distribuição por Dia da Semana</h3>
      <div class="weekday-chart">
        {#each ($analyticsData.trends?.weekday || []) as day}
          <div class="weekday-item">
            <span class="weekday-name">{day.name}</span>
            <div class="weekday-bar-container">
              <div class="weekday-bar" style="width: {getBarWidth(day.count, maxWeekday)}%"></div>
            </div>
            <span class="weekday-count">{day.count}</span>
          </div>
        {/each}
      </div>
    </div>

  {:else}
    <div class="no-data">
      Nenhum dado disponível
    </div>
  {/if}
</div>

<style>
  .analytics-panel {
    padding: 16px;
    overflow-y: auto;
    height: 100%;
    background: var(--bg-secondary, #1a1a2e);
    color: var(--text-primary, #e0e0e0);
  }

  .period-selector {
    display: flex;
    gap: 8px;
    margin-bottom: 20px;
    flex-wrap: wrap;
  }

  .period-btn {
    padding: 6px 12px;
    border: 1px solid var(--border-color, #333);
    border-radius: 4px;
    background: var(--bg-tertiary, #252540);
    color: var(--text-secondary, #aaa);
    cursor: pointer;
    font-size: 12px;
    transition: all 0.2s;
  }

  .period-btn:hover {
    background: var(--bg-hover, #303050);
  }

  .period-btn.active {
    background: var(--accent-color, #4ecdc4);
    color: var(--bg-primary, #0f0f1a);
    border-color: var(--accent-color, #4ecdc4);
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 40px;
    color: var(--text-secondary, #aaa);
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid var(--border-color, #333);
    border-top-color: var(--accent-color, #4ecdc4);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .overview-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
    gap: 12px;
    margin-bottom: 24px;
  }

  .card {
    background: var(--bg-tertiary, #252540);
    border-radius: 8px;
    padding: 16px;
    text-align: center;
  }

  .card.unread {
    border-left: 3px solid var(--accent-color, #4ecdc4);
  }

  .card-value {
    font-size: 24px;
    font-weight: bold;
    color: var(--text-primary, #e0e0e0);
    margin-bottom: 4px;
  }

  .card-label {
    font-size: 11px;
    color: var(--text-secondary, #aaa);
    text-transform: uppercase;
  }

  .section {
    margin-bottom: 24px;
  }

  .section h3 {
    font-size: 14px;
    color: var(--text-secondary, #aaa);
    margin-bottom: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .response-stats {
    display: flex;
    gap: 24px;
  }

  .stat {
    display: flex;
    flex-direction: column;
  }

  .stat-value {
    font-size: 20px;
    font-weight: bold;
    color: var(--accent-color, #4ecdc4);
  }

  .stat-label {
    font-size: 11px;
    color: var(--text-secondary, #aaa);
  }

  .top-senders {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .sender-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px;
    background: var(--bg-tertiary, #252540);
    border-radius: 6px;
  }

  .sender-rank {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary, #1a1a2e);
    border-radius: 50%;
    font-size: 12px;
    font-weight: bold;
    color: var(--text-secondary, #aaa);
  }

  .sender-info {
    flex: 1;
    min-width: 0;
  }

  .sender-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary, #e0e0e0);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sender-email {
    font-size: 11px;
    color: var(--text-secondary, #aaa);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sender-stats {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 120px;
  }

  .sender-bar-container {
    flex: 1;
    height: 6px;
    background: var(--bg-secondary, #1a1a2e);
    border-radius: 3px;
    overflow: hidden;
  }

  .sender-bar {
    height: 100%;
    background: var(--accent-color, #4ecdc4);
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .sender-count {
    font-size: 12px;
    font-weight: bold;
    color: var(--text-primary, #e0e0e0);
    min-width: 30px;
    text-align: right;
  }

  .sender-unread {
    font-size: 10px;
    color: var(--accent-color, #4ecdc4);
  }

  .hourly-chart {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    height: 80px;
    padding-bottom: 20px;
    position: relative;
  }

  .hour-bar-container {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-end;
    position: relative;
  }

  .hour-bar {
    width: 100%;
    background: var(--accent-color, #4ecdc4);
    border-radius: 2px 2px 0 0;
    min-height: 2px;
    transition: height 0.3s ease;
  }

  .hour-label {
    position: absolute;
    bottom: -18px;
    font-size: 10px;
    color: var(--text-secondary, #aaa);
  }

  .weekday-chart {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .weekday-item {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .weekday-name {
    width: 32px;
    font-size: 12px;
    color: var(--text-secondary, #aaa);
  }

  .weekday-bar-container {
    flex: 1;
    height: 16px;
    background: var(--bg-tertiary, #252540);
    border-radius: 4px;
    overflow: hidden;
  }

  .weekday-bar {
    height: 100%;
    background: linear-gradient(90deg, var(--accent-color, #4ecdc4), var(--accent-secondary, #45b7aa));
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .weekday-count {
    width: 40px;
    font-size: 12px;
    color: var(--text-primary, #e0e0e0);
    text-align: right;
  }

  .no-data {
    padding: 40px;
    text-align: center;
    color: var(--text-secondary, #aaa);
  }
</style>
