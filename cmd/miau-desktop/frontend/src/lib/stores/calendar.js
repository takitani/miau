import { writable, derived, get } from 'svelte/store';
import {
  GetCalendarEvents,
  GetCalendarEventsForWeek,
  GetUpcomingCalendarEvents,
  CreateCalendarEvent,
  UpdateCalendarEvent,
  DeleteCalendarEvent,
  ToggleCalendarEventComplete,
  GetCalendarEventCounts,
  CreateFollowUpEvent,
  SyncTasksToCalendar,
  IsGoogleCalendarConnected,
  ListGoogleCalendars,
  SyncFromGoogleCalendar,
  GetGoogleCalendarEvents
} from '../wailsjs/wailsjs/go/desktop/App.js';

// Current week start date (Monday)
export const currentWeekStart = writable(getWeekStart(new Date()));

// Events for current view
export const calendarEvents = writable([]);

// Loading state
export const calendarLoading = writable(false);

// Selected event for modal
export const selectedEvent = writable(null);

// Show event modal
export const showEventModal = writable(false);

// Show calendar panel
export const showCalendarPanel = writable(false);

// Event counts
export const eventCounts = writable({ upcoming: 0, completed: 0, total: 0 });

// Event type colors
export const eventTypeColors = {
  custom: '#4ecdc4',        // Teal
  task_deadline: '#f39c12', // Orange
  email_followup: '#9b59b6', // Purple
  meeting: '#3498db'        // Blue
};

// Event type labels
export const eventTypeLabels = {
  custom: 'Event',
  task_deadline: 'Task',
  email_followup: 'Follow-up',
  meeting: 'Meeting'
};

// === Helper Functions ===

// Get Monday of the week containing the given date
export function getWeekStart(date) {
  const d = new Date(date);
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1); // Adjust for Sunday
  d.setDate(diff);
  d.setHours(0, 0, 0, 0);
  return d;
}

// Format date as YYYY-MM-DD for API calls
export function formatDateForAPI(date) {
  return date.toISOString().split('T')[0];
}

// Get day name
export function getDayName(date) {
  return date.toLocaleDateString('pt-BR', { weekday: 'short' });
}

// Get short date format
export function formatShortDate(date) {
  return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' });
}

// Get time string
export function formatTime(date) {
  return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
}

// === Data Loading ===

// Load events for the current week
export async function loadWeekEvents(weekStart = null) {
  if (!weekStart)
    weekStart = get(currentWeekStart);

  calendarLoading.set(true);
  try {
    const dateStr = formatDateForAPI(weekStart);
    const result = await GetCalendarEventsForWeek(dateStr);
    calendarEvents.set(result || []);
    await loadEventCounts();
  } catch (err) {
    console.error('Failed to load week events:', err);
    calendarEvents.set([]);
  } finally {
    calendarLoading.set(false);
  }
}

// Load upcoming events
export async function loadUpcomingEvents(limit = 10) {
  calendarLoading.set(true);
  try {
    const result = await GetUpcomingCalendarEvents(limit);
    return result || [];
  } catch (err) {
    console.error('Failed to load upcoming events:', err);
    return [];
  } finally {
    calendarLoading.set(false);
  }
}

// Load event counts
export async function loadEventCounts() {
  try {
    const counts = await GetCalendarEventCounts();
    if (counts)
      eventCounts.set(counts);
  } catch (err) {
    console.error('Failed to load event counts:', err);
  }
}

// === Navigation ===

// Navigate to previous week
export function previousWeek() {
  currentWeekStart.update(date => {
    const newDate = new Date(date);
    newDate.setDate(newDate.getDate() - 7);
    return newDate;
  });
  loadWeekEvents();
}

// Navigate to next week
export function nextWeek() {
  currentWeekStart.update(date => {
    const newDate = new Date(date);
    newDate.setDate(newDate.getDate() + 7);
    return newDate;
  });
  loadWeekEvents();
}

// Go to today's week
export function goToToday() {
  currentWeekStart.set(getWeekStart(new Date()));
  loadWeekEvents();
}

// === CRUD Operations ===

// Create a new event
export async function createEvent(input) {
  try {
    const newEvent = await CreateCalendarEvent(input);
    if (newEvent) {
      calendarEvents.update(list => [...list, newEvent].sort((a, b) =>
        new Date(a.startTime) - new Date(b.startTime)
      ));
      await loadEventCounts();
    }
    return newEvent;
  } catch (err) {
    console.error('Failed to create event:', err);
    throw err;
  }
}

// Update an existing event
export async function updateEvent(id, updates) {
  try {
    const currentEvents = get(calendarEvents);
    const existing = currentEvents.find(e => e.id === id);
    if (!existing)
      throw new Error('Event not found');

    const input = {
      id,
      title: updates.title ?? existing.title,
      description: updates.description ?? existing.description,
      eventType: updates.eventType ?? existing.eventType,
      startTime: updates.startTime ?? existing.startTime,
      endTime: updates.endTime ?? existing.endTime,
      allDay: updates.allDay ?? existing.allDay,
      color: updates.color ?? existing.color,
      taskId: updates.taskId ?? existing.taskId,
      emailId: updates.emailId ?? existing.emailId,
      isCompleted: updates.isCompleted ?? existing.isCompleted,
      source: updates.source ?? existing.source
    };

    const updated = await UpdateCalendarEvent(input);
    if (updated) {
      calendarEvents.update(list =>
        list.map(e => e.id === id ? updated : e)
          .sort((a, b) => new Date(a.startTime) - new Date(b.startTime))
      );
      await loadEventCounts();
    }
    return updated;
  } catch (err) {
    console.error('Failed to update event:', err);
    throw err;
  }
}

// Toggle event completion
export async function toggleEventComplete(id) {
  try {
    const newStatus = await ToggleCalendarEventComplete(id);
    calendarEvents.update(list => list.map(e => {
      if (e.id === id)
        return { ...e, isCompleted: newStatus };
      return e;
    }));
    await loadEventCounts();
    return newStatus;
  } catch (err) {
    console.error('Failed to toggle event:', err);
    throw err;
  }
}

// Delete an event
export async function deleteEvent(id) {
  try {
    await DeleteCalendarEvent(id);
    calendarEvents.update(list => list.filter(e => e.id !== id));
    await loadEventCounts();
  } catch (err) {
    console.error('Failed to delete event:', err);
    throw err;
  }
}

// === Email Follow-up ===

// Create follow-up event for an email
export async function createFollowUp(emailId, followUpDate, title = '') {
  try {
    const event = await CreateFollowUpEvent(emailId, followUpDate, title);
    if (event) {
      calendarEvents.update(list => [...list, event].sort((a, b) =>
        new Date(a.startTime) - new Date(b.startTime)
      ));
      await loadEventCounts();
    }
    return event;
  } catch (err) {
    console.error('Failed to create follow-up:', err);
    throw err;
  }
}

// === Task Sync ===

// Sync all tasks with due dates to calendar
export async function syncTasks() {
  try {
    await SyncTasksToCalendar();
    await loadWeekEvents();
  } catch (err) {
    console.error('Failed to sync tasks:', err);
    throw err;
  }
}

// === Modal Helpers ===

// Open event modal for creating new event
export function openCreateEventModal(date = null) {
  selectedEvent.set({
    id: null,
    title: '',
    description: '',
    eventType: 'custom',
    startTime: date || new Date(),
    endTime: null,
    allDay: false,
    color: eventTypeColors.custom
  });
  showEventModal.set(true);
}

// Open event modal for editing
export function openEditEventModal(event) {
  selectedEvent.set({ ...event });
  showEventModal.set(true);
}

// Close event modal
export function closeEventModal() {
  showEventModal.set(false);
  selectedEvent.set(null);
}

// === Derived Stores ===

// Events grouped by day of week (for calendar view)
export const eventsByDay = derived(
  [calendarEvents, currentWeekStart],
  ([$events, $weekStart]) => {
    const days = {};
    for (let i = 0; i < 7; i++) {
      const date = new Date($weekStart);
      date.setDate(date.getDate() + i);
      const key = formatDateForAPI(date);
      days[key] = {
        date: date,
        dayName: getDayName(date),
        shortDate: formatShortDate(date),
        isToday: formatDateForAPI(date) === formatDateForAPI(new Date()),
        events: []
      };
    }

    // Group events into days
    for (const event of $events) {
      const eventDate = new Date(event.startTime);
      const key = formatDateForAPI(eventDate);
      if (days[key])
        days[key].events.push(event);
    }

    return Object.values(days);
  }
);

// Week label for header (e.g., "Dec 2 - 8, 2025")
export const weekLabel = derived(currentWeekStart, ($weekStart) => {
  const endDate = new Date($weekStart);
  endDate.setDate(endDate.getDate() + 6);

  const startMonth = $weekStart.toLocaleDateString('pt-BR', { month: 'short' });
  const endMonth = endDate.toLocaleDateString('pt-BR', { month: 'short' });

  if (startMonth === endMonth) {
    return `${$weekStart.getDate()} - ${endDate.getDate()} ${startMonth} ${endDate.getFullYear()}`;
  } else {
    return `${$weekStart.getDate()} ${startMonth} - ${endDate.getDate()} ${endMonth} ${endDate.getFullYear()}`;
  }
});

// Is current week the "today" week?
export const isCurrentWeek = derived(currentWeekStart, ($weekStart) => {
  const todayWeekStart = getWeekStart(new Date());
  return formatDateForAPI($weekStart) === formatDateForAPI(todayWeekStart);
});

// === Google Calendar Integration ===

// Google Calendar connection status
export const googleCalendarConnected = writable(false);

// Available Google Calendars
export const googleCalendars = writable([]);

// Selected Google Calendar for sync
export const selectedGoogleCalendar = writable('primary');

// Google Calendar sync loading state
export const googleSyncLoading = writable(false);

// Check if Google Calendar is connected
export async function checkGoogleCalendarConnection() {
  try {
    const connected = await IsGoogleCalendarConnected();
    googleCalendarConnected.set(connected);
    return connected;
  } catch (err) {
    console.error('Failed to check Google Calendar connection:', err);
    googleCalendarConnected.set(false);
    return false;
  }
}

// Load available Google Calendars
export async function loadGoogleCalendars() {
  try {
    const calendars = await ListGoogleCalendars();
    googleCalendars.set(calendars || []);
    return calendars || [];
  } catch (err) {
    console.error('Failed to load Google Calendars:', err);
    googleCalendars.set([]);
    return [];
  }
}

// Sync events from Google Calendar
export async function syncFromGoogle(calendarId = 'primary') {
  googleSyncLoading.set(true);
  try {
    const count = await SyncFromGoogleCalendar(calendarId);
    // Reload local events after sync
    await loadWeekEvents();
    return count;
  } catch (err) {
    console.error('Failed to sync from Google Calendar:', err);
    throw err;
  } finally {
    googleSyncLoading.set(false);
  }
}

// Get events directly from Google Calendar (without saving locally)
export async function getGoogleEvents(calendarId, weekStart) {
  try {
    const dateStr = formatDateForAPI(weekStart || get(currentWeekStart));
    return await GetGoogleCalendarEvents(calendarId || 'primary', dateStr);
  } catch (err) {
    console.error('Failed to get Google Calendar events:', err);
    return [];
  }
}
