import { computed, inject, Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { finalize, forkJoin } from 'rxjs';

import { Alert, DashboardSnapshot, Device, DnsEvent, FlowEvent } from './models';

@Injectable({ providedIn: 'root' })
export class DashboardApiService {
  private readonly http = inject(HttpClient);

  readonly devices = signal<Device[]>([]);
  readonly alerts = signal<Alert[]>([]);
  readonly dnsEvents = signal<DnsEvent[]>([]);
  readonly flowEvents = signal<FlowEvent[]>([]);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);
  readonly lastLoadedAt = signal<Date | null>(null);

  readonly summary = computed(() => {
    const devices = this.devices();
    const alerts = this.alerts();
    const dnsEvents = this.dnsEvents();
    const flowEvents = this.flowEvents();

    return {
      totalDevices: devices.length,
      childProfiles: devices.filter((device) => device.profile_id === 'child').length,
      iotDevices: devices.filter((device) => ['iot', 'camera'].includes(device.device_type)).length,
      riskyDevices: devices.filter((device) => device.risk_score >= 50).length,
      openAlerts: alerts.filter((alert) => alert.status === 'open').length,
      highAlerts: alerts.filter((alert) => ['high', 'critical'].includes(alert.severity)).length,
      bypassAlerts: alerts.filter((alert) => alert.type === 'dns_bypass').length,
      policyAlerts: alerts.filter((alert) => alert.type === 'policy_violation').length,
      dnsQueries: dnsEvents.length,
      flows: flowEvents.length
    };
  });

  refresh(): void {
    this.loading.set(true);
    this.error.set(null);

    forkJoin({
      devices: this.http.get<Device[] | null>('/api/devices'),
      alerts: this.http.get<Alert[] | null>('/api/alerts'),
      dnsEvents: this.http.get<DnsEvent[] | null>('/api/activity/dns'),
      flowEvents: this.http.get<FlowEvent[] | null>('/api/activity/flows')
    })
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (snapshot) => this.applySnapshot(snapshot),
        error: (error: unknown) => {
          this.error.set(this.messageFromError(error, 'Nao foi possivel carregar os dados do painel.'));
        }
      });
  }

  updateProfile(deviceId: string, profileId: string): void {
    this.http
      .post<Device>(`/api/devices/${deviceId}/profile`, { profile_id: profileId })
      .subscribe({
        next: (device) => {
          this.devices.update((devices) =>
            devices.map((current) => (current.id === device.id ? device : current))
          );
        },
        error: (error: unknown) => {
          this.error.set(this.messageFromError(error, 'Nao foi possivel atualizar o perfil.'));
        }
      });
  }

  acknowledgeAlert(alertId: string): void {
    this.http.post<Alert>(`/api/alerts/${alertId}/ack`, {}).subscribe({
      next: (alert) => {
        this.alerts.update((alerts) =>
          alerts.map((current) => (current.id === alert.id ? alert : current))
        );
      },
      error: (error: unknown) => {
        this.error.set(this.messageFromError(error, 'Nao foi possivel reconhecer o alerta.'));
      }
    });
  }

  private applySnapshot(snapshot: DashboardSnapshot): void {
    this.devices.set(snapshot.devices ?? []);
    this.alerts.set(snapshot.alerts ?? []);
    this.dnsEvents.set(snapshot.dnsEvents ?? []);
    this.flowEvents.set(snapshot.flowEvents ?? []);
    this.lastLoadedAt.set(new Date());
  }

  private messageFromError(error: unknown, fallback: string): string {
    if (error instanceof Error && error.message) {
      return error.message;
    }

    return fallback;
  }
}

