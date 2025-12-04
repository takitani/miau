import { writable, get } from 'svelte/store';
import { info, error as logError, debug as logDebug } from './debug.js';

// Analytics data
export const analyticsData = writable(null);

// Loading state
export const analyticsLoading = writable(false);

// Selected period
export const analyticsPeriod = writable('30d');

// Load analytics from backend
export async function loadAnalytics(period = '30d') {
  analyticsLoading.set(true);
  analyticsPeriod.set(period);
  logDebug(`loadAnalytics called: period=${period}`);

  try {
    if (typeof window !== 'undefined' && window.go && window.go.desktop && window.go.desktop.App) {
      logDebug('Calling Go backend GetAnalytics...');
      const result = await window.go.desktop.App.GetAnalytics(period);
      logDebug('GetAnalytics returned:', result);
      analyticsData.set(result);
      info(`Analytics loaded for period ${period}`);
    } else {
      // Mock data for development
      logDebug('Wails bindings not available, using mock data');
      analyticsData.set(getMockAnalytics());
    }
  } catch (err) {
    logError('Failed to load analytics', err);
    analyticsData.set(null);
  } finally {
    analyticsLoading.set(false);
  }
}

// Get overview only
export async function loadOverview() {
  try {
    if (window.go?.desktop?.App) {
      const result = await window.go.desktop.App.GetAnalyticsOverview();
      return result;
    }
    return getMockAnalytics().overview;
  } catch (err) {
    logError('Failed to load overview', err);
    return null;
  }
}

// Get top senders
export async function loadTopSenders(limit = 10, period = '30d') {
  try {
    if (window.go?.desktop?.App) {
      const result = await window.go.desktop.App.GetTopSenders(limit, period);
      return result;
    }
    return getMockAnalytics().topSenders;
  } catch (err) {
    logError('Failed to load top senders', err);
    return [];
  }
}

// Mock data for development
function getMockAnalytics() {
  return {
    overview: {
      totalEmails: 1234,
      unreadEmails: 45,
      starredEmails: 12,
      archivedEmails: 567,
      sentEmails: 89,
      draftCount: 3,
      storageUsedMb: 156.7
    },
    topSenders: [
      { email: 'notifications@github.com', name: 'GitHub', count: 234, unreadCount: 12, percentage: 25.5 },
      { email: 'newsletter@medium.com', name: 'Medium', count: 189, unreadCount: 45, percentage: 20.6 },
      { email: 'john@work.com', name: 'John Smith', count: 112, unreadCount: 3, percentage: 12.2 },
      { email: 'maria@example.com', name: 'Maria Silva', count: 87, unreadCount: 8, percentage: 9.5 },
      { email: 'no-reply@amazon.com', name: 'Amazon', count: 65, unreadCount: 2, percentage: 7.1 },
      { email: 'team@slack.com', name: 'Slack', count: 54, unreadCount: 0, percentage: 5.9 },
      { email: 'noreply@google.com', name: 'Google', count: 43, unreadCount: 1, percentage: 4.7 },
      { email: 'support@company.com', name: 'Support', count: 32, unreadCount: 5, percentage: 3.5 },
      { email: 'newsletter@dev.to', name: 'DEV', count: 28, unreadCount: 8, percentage: 3.1 },
      { email: 'updates@linkedin.com', name: 'LinkedIn', count: 21, unreadCount: 4, percentage: 2.3 }
    ],
    trends: {
      daily: [
        { date: '2024-12-04', count: 45 },
        { date: '2024-12-03', count: 52 },
        { date: '2024-12-02', count: 38 },
        { date: '2024-12-01', count: 28 },
        { date: '2024-11-30', count: 15 },
        { date: '2024-11-29', count: 41 },
        { date: '2024-11-28', count: 36 }
      ],
      hourly: Array.from({ length: 24 }, (_, i) => ({
        hour: i,
        count: Math.floor(Math.random() * 50) + (i >= 9 && i <= 18 ? 30 : 5)
      })),
      weekday: [
        { weekday: 0, name: 'Dom', count: 45 },
        { weekday: 1, name: 'Seg', count: 120 },
        { weekday: 2, name: 'Ter', count: 135 },
        { weekday: 3, name: 'Qua', count: 142 },
        { weekday: 4, name: 'Qui', count: 128 },
        { weekday: 5, name: 'Sex', count: 98 },
        { weekday: 6, name: 'Sáb', count: 32 }
      ]
    },
    responseTime: {
      avgResponseMinutes: 247.5,
      responseRate: 34.2
    },
    period: '30d',
    generatedAt: new Date().toISOString()
  };
}
