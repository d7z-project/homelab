import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'ua',
  standalone: true,
})
export class UaPipe implements PipeTransform {
  transform(value: string | undefined): string {
    if (!value) return '-';
    if (value.includes('Firefox/')) return 'Firefox';
    if (value.includes('Edg/')) return 'Edge';
    if (value.includes('Chrome/')) return 'Chrome';
    if (value.includes('Safari/')) return 'Safari';
    if (value.includes('Postman')) return 'Postman';
    return value.length > 20 ? value.substring(0, 20) + '...' : value;
  }
}
