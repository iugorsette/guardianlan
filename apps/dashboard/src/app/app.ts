import { CommonModule, DatePipe, DecimalPipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  signal
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { interval, startWith } from 'rxjs';

import { DashboardApiService } from './dashboard-api.service';
import { Alert, Device } from './models';
import { DeviceEvidence, DeviceInsight } from './models';
import { DASHBOARD_AUTO_REFRESH_MS } from './dashboard.tokens';

@Component({
  selector: 'app-root',
  imports: [CommonModule, FormsModule, DatePipe, DecimalPipe],
  templateUrl: './app.html',
  styleUrl: './app.css',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class App {
  protected readonly title = 'Guardian LAN';
  protected readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly autoRefreshMs = inject(DASHBOARD_AUTO_REFRESH_MS);
  protected readonly profileDrafts = signal<Record<string, string>>({});
  protected readonly nameDrafts = signal<Record<string, string>>({});
  protected readonly summary = this.api.summary;
  protected readonly devices = computed(() =>
    [...this.api.devices()].sort((left, right) => right.risk_score - left.risk_score)
  );
  protected readonly alerts = computed(() =>
    [...this.api.alerts()].sort(
      (left, right) =>
        new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
    )
  );
  protected readonly dnsEvents = computed(() => this.api.dnsEvents().slice(0, 8));
  protected readonly flowEvents = computed(() => this.api.flowEvents().slice(0, 8));
  protected readonly cameraDevices = computed(() =>
    this.devices().filter((device) => device.device_type === 'camera')
  );

  constructor() {
    if (this.autoRefreshMs < 0) {
      return;
    }

    interval(this.autoRefreshMs)
      .pipe(startWith(0), takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.api.refresh());
  }

  protected trackById = (_: number, item: { id: string }) => item.id;

  protected profileFor(device: Device): string {
    return this.profileDrafts()[device.id] ?? device.profile_id;
  }

  protected nameFor(device: Device): string {
    return this.nameDrafts()[device.id] ?? device.display_name ?? '';
  }

  protected setProfile(deviceId: string, profileId: string): void {
    this.profileDrafts.update((drafts) => ({ ...drafts, [deviceId]: profileId }));
  }

  protected setDeviceName(deviceId: string, displayName: string): void {
    this.nameDrafts.update((drafts) => ({ ...drafts, [deviceId]: displayName }));
  }

  protected saveProfile(device: Device): void {
    const profileId = this.profileFor(device);
    if (profileId === device.profile_id) {
      return;
    }

    this.api.updateProfile(device.id, profileId);
  }

  protected saveDeviceName(device: Device): void {
    const displayName = this.nameFor(device).trim();
    if (displayName === (device.display_name ?? '')) {
      return;
    }

    this.api.updateDeviceName(device.id, displayName);
  }

  protected acknowledge(alert: Alert): void {
    if (alert.status !== 'open') {
      return;
    }

    this.api.acknowledgeAlert(alert.id);
  }

  protected latestInsight(deviceId: string): DeviceInsight | null {
    return this.api.deviceInsights()[deviceId]?.[0] ?? null;
  }

  protected evidenceFor(deviceId: string): DeviceEvidence | null {
    const insight = this.latestInsight(deviceId);
    if (!insight) {
      return null;
    }

    return insight.evidence as DeviceEvidence;
  }
}
