import { Component, OnInit, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [],
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent implements OnInit {
  message = signal('');
  loading = signal(true);

  constructor(private http: HttpClient) {}

  ngOnInit() {
    this.http.get<{ message: string }>('/api/hello').subscribe({
      next: (data) => {
        this.message.set(data.message);
        this.loading.set(false);
      },
      error: () => {
        this.message.set('Failed to load message');
        this.loading.set(false);
      }
    });
  }
}
