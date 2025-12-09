import { startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval, format, isSameMonth, isSameDay } from 'date-fns';

export function generateMonthGrid(date: Date): Date[] {
    const monthStart = startOfMonth(date);
    const monthEnd = endOfMonth(date);

    const startDate = startOfWeek(monthStart);
    const endDate = endOfWeek(monthEnd);

    return eachDayOfInterval({
        start: startDate,
        end: endDate
    });
}

export function formatMonthYear(date: Date): string {
    return format(date, 'MMMM yyyy');
}

export function isSameDate(d1: Date, d2: Date): boolean {
    return isSameDay(d1, d2);
}

export function isCurrentMonth(d: Date, monthDate: Date): boolean {
    return isSameMonth(d, monthDate);
}
