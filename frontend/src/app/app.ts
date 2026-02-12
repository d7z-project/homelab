import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ExampleService } from './generated';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: `
    <h1>Hello, {{ title() }}</h1>

    <router-outlet />
  `,
  styles: [],
})
export class App implements OnInit {
  protected readonly title = signal('frontend');
  private exampleService = inject(ExampleService);

  ngOnInit() {
    this.exampleService.pingGet().subscribe({
      next: (res) => console.log('Ping success:', res),
      error: (err) => console.error('Ping failed:', err)
    });
  }
}
