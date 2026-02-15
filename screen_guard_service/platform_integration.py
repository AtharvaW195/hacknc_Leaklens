"""
Platform-Specific Integrations for Screen Share Monitoring
Zoom, Teams, Google Meet, Discord, etc.
"""

import subprocess
try:
    import psutil
except ImportError:
    psutil = None
import time
from typing import Optional, List, Dict, Any
from realtime_monitor import RealTimeScreenShareMonitor


class ScreenShareDetector:
    """
    Automatically detect when screen sharing is active
    and start/stop monitoring accordingly
    """
    
    SCREEN_SHARE_PROCESSES = {
        'zoom': ['zoom', 'zoom.us', 'CptHost.exe'],
        'teams': ['Teams.exe', 'ms-teams', 'teams'],
        'meet': ['chrome', 'firefox', 'safari'],  # Browser-based
        'discord': ['Discord.exe', 'discord'],
        'slack': ['slack', 'Slack.exe'],
        'webex': ['CiscoCollabHost', 'webex', 'ptoneclk'],
    }
    
    def __init__(self, platforms: Optional[List[str]] = None):
        if psutil is None:
            raise RuntimeError(
                "psutil is required for auto platform detection. "
                "Install it with: pip install psutil"
            )
        self.is_sharing = False
        self.active_platform = None
        self.cpu_share_threshold = 4.0
        self.cpu_sample_window_seconds = 0.2
        normalized = [p.strip().lower() for p in (platforms or []) if p.strip()]
        valid = [p for p in normalized if p in self.SCREEN_SHARE_PROCESSES]
        self.platforms = valid if valid else list(self.SCREEN_SHARE_PROCESSES.keys())
    
    def is_process_running(self, process_names: List[str]) -> bool:
        """Check if any of the given processes are running"""
        for proc in psutil.process_iter(['name']):
            try:
                proc_name = (proc.info['name'] or '').lower()
                if any(name.lower() in proc_name for name in process_names):
                    return True
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue
        return False

    def is_process_using_screen_capture(self, process_names: List[str], cpu_threshold: Optional[float] = None) -> bool:
        """
        Heuristic: active screen sharing usually keeps process CPU above idle baseline.
        """
        threshold = cpu_threshold if cpu_threshold is not None else self.cpu_share_threshold
        matching = []
        for proc in psutil.process_iter(['name']):
            try:
                proc_name = (proc.info['name'] or '').lower()
                if any(name.lower() in proc_name for name in process_names):
                    matching.append(proc)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue

        if not matching:
            return False

        # Warm up psutil CPU counters, then sample after a short window.
        for proc in matching:
            try:
                proc.cpu_percent(interval=None)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue

        time.sleep(self.cpu_sample_window_seconds)

        total_cpu = 0.0
        for proc in matching:
            try:
                total_cpu += proc.cpu_percent(interval=None)
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue

        return total_cpu >= threshold
    
    def detect_active_platform(self) -> Optional[str]:
        """Detect which screen sharing platform is active"""
        for platform in self.platforms:
            processes = self.SCREEN_SHARE_PROCESSES[platform]
            if self.is_process_running(processes):
                return platform
        return None
    
    def is_screen_sharing_active(self) -> bool:
        """
        Detect if screen sharing is currently active
        Platform-specific detection
        """
        platform = self.detect_active_platform()
        
        if not platform:
            return False
        
        # Platform-specific detection logic
        if platform == 'zoom':
            return self._is_zoom_sharing()
        elif platform == 'teams':
            return self._is_teams_sharing()
        elif platform == 'discord':
            return self._is_discord_sharing()
        elif platform == 'slack':
            return self._is_slack_sharing()
        elif platform in ('meet', 'webex'):
            return self.is_process_using_screen_capture(self.SCREEN_SHARE_PROCESSES[platform], cpu_threshold=3.0)

        return False
    
    def _is_zoom_sharing(self) -> bool:
        """Detect if Zoom is actively sharing screen"""
        # Check for Zoom's screen share window
        # On macOS: check for "zoom share toolbar"
        # On Windows: check for CptHost.exe process
        # This is simplified - real implementation would check window titles
        
        try:
            # Check for Zoom share helper processes (cross-platform hints).
            for proc in psutil.process_iter(['name']):
                name = (proc.info.get('name') or '').lower()
                if any(marker in name for marker in ('cpthost', 'zoomcptservice', 'zoom share')):
                    return True

            # Check command-line hints on zoom processes.
            for proc in psutil.process_iter(['name', 'cmdline']):
                name = (proc.info.get('name') or '').lower()
                if 'zoom' not in name:
                    continue
                cmdline = ' '.join(proc.info.get('cmdline', [])).lower()
                if any(marker in cmdline for marker in ('share', 'screen', 'cpthost')):
                    return True

            # CPU-based fallback: threshold intentionally low, summed across zoom processes.
            return self.is_process_using_screen_capture(['zoom', 'zoom.us'], cpu_threshold=1.5)
        except:
            pass
        
        return False
    
    def _is_teams_sharing(self) -> bool:
        """Detect if Teams is actively sharing screen"""
        # Check for Teams sharing process
        # Teams creates a separate process when sharing
        try:
            for proc in psutil.process_iter(['name', 'cmdline']):
                name = (proc.info['name'] or '').lower()
                if 'teams' in name:
                    # Check command line for share indicators
                    cmdline = ' '.join(proc.info.get('cmdline', [])).lower()
                    if 'share' in cmdline or 'screenshare' in cmdline:
                        return True
        except:
            pass
        
        return False
    
    def _is_discord_sharing(self) -> bool:
        """Detect if Discord is actively sharing screen"""
        try:
            return self.is_process_using_screen_capture(['discord', 'Discord.exe'], cpu_threshold=4.0)
        except:
            pass
        
        return False

    def _is_slack_sharing(self) -> bool:
        """Detect if Slack is actively sharing screen in huddles/calls."""
        try:
            # Signal 1: Slack + renderer helper with share/call hints in cmdline.
            for proc in psutil.process_iter(['name', 'cmdline']):
                name = (proc.info.get('name') or '').lower()
                if 'slack' not in name:
                    continue
                cmdline = ' '.join(proc.info.get('cmdline', [])).lower()
                share_markers = ('screenshare', 'screen-share', 'huddle', 'call', 'webrtc')
                if any(marker in cmdline for marker in share_markers):
                    return True

            # Signal 2: CPU activity on Slack processes while app is present.
            return self.is_process_using_screen_capture(
                ['slack', 'slack helper', 'slack.exe'],
                cpu_threshold=1.0
            )
        except:
            pass
        return False


class AutoScreenShareMonitor:
    """
    Automatically starts monitoring when screen sharing begins
    and stops when it ends
    """
    
    def __init__(self,
                 scan_interval: float = 2.0,
                 check_interval: float = 5.0,
                 use_ml: bool = True,
                 use_gpu: bool = True,
                 alert_callback = None,
                 platforms: Optional[List[str]] = None,
                 start_confirmations: int = 2,
                 stop_confirmations: int = 2,
                 monitor_kwargs: Optional[Dict[str, Any]] = None):
        """
        Initialize auto monitor
        
        Args:
            scan_interval: How often to scan screen
            check_interval: How often to check if sharing is active
            use_ml: Use ML scanner
            use_gpu: Use GPU acceleration
            alert_callback: Function to call on detection
            platforms: Platform allow-list (e.g., ["zoom", "slack"])
            start_confirmations: Consecutive positive checks needed to start
            stop_confirmations: Consecutive negative checks needed to stop
            monitor_kwargs: Additional kwargs passed to RealTimeScreenShareMonitor
        """
        
        self.check_interval = check_interval
        self.start_confirmations = max(1, start_confirmations)
        self.stop_confirmations = max(1, stop_confirmations)
        self.share_positive_count = 0
        self.share_negative_count = 0
        self.monitor_kwargs = monitor_kwargs or {}
        
        # Initialize detector
        self.detector = ScreenShareDetector(platforms=platforms)
        
        # Initialize monitor (but don't start yet)
        self.monitor = RealTimeScreenShareMonitor(
            scan_interval=scan_interval,
            use_ml=use_ml,
            use_gpu=use_gpu,
            alert_callback=alert_callback,
            **self.monitor_kwargs
        )
        
        self.is_running = False
        self.was_sharing = False
    
    def run(self):
        """Main loop - automatically start/stop monitoring"""
        print("\n" + "="*80)
        print("🤖 AUTO SCREEN SHARE MONITOR")
        print("="*80)
        print("\nAutomatically monitors when you start screen sharing")
        print(f"Watching platforms: {', '.join(self.detector.platforms)}")
        print("\nPress Ctrl+C to stop\n")
        
        self.is_running = True
        
        try:
            while self.is_running:
                # Check if screen sharing is active
                is_sharing = self.detector.is_screen_sharing_active()
                platform = self.detector.detect_active_platform()

                if is_sharing:
                    self.share_positive_count += 1
                    self.share_negative_count = 0
                else:
                    self.share_negative_count += 1
                    self.share_positive_count = 0
                
                # State changed: started sharing
                if (not self.was_sharing) and self.share_positive_count >= self.start_confirmations:
                    print(f"\n📹 Screen sharing detected ({platform})!")
                    print("🔍 Starting monitor...\n")
                    self.monitor.start_monitoring()
                    self.was_sharing = True
                
                # State changed: stopped sharing
                elif self.was_sharing and self.share_negative_count >= self.stop_confirmations:
                    print(f"\n📵 Screen sharing stopped")
                    print("🛑 Stopping monitor...\n")
                    self.monitor.stop_monitoring()
                    self.was_sharing = False
                
                # Wait before checking again
                time.sleep(self.check_interval)
                
        except KeyboardInterrupt:
            print("\n\n⚠️  Shutting down...")
            if self.was_sharing:
                self.monitor.stop_monitoring()
            print("✅ Stopped")


# ========================================
# ZOOM INTEGRATION
# ========================================

class ZoomMonitorIntegration:
    """
    Integration specifically for Zoom
    Can hook into Zoom's SDK or use screen capture
    """
    
    def __init__(self):
        self.monitor = RealTimeScreenShareMonitor(
            scan_interval=2.0,
            use_ml=True,
            alert_callback=self.zoom_alert
        )
    
    def zoom_alert(self, results):
        """Custom alert for Zoom"""
        detections = results['detections']
        if not detections:
            return
        
        critical = sum(1 for d in detections if getattr(d, 'severity', '') == 'critical')
        
        if critical > 0:
            print("\n" + "="*80)
            print("🚨 ZOOM ALERT - CRITICAL CONTENT DETECTED!")
            print("="*80)
            print("\n⚠️  RECOMMENDATION: Stop screen share immediately")
            print("   Click 'Stop Share' in Zoom controls")
            print("\n" + "="*80 + "\n")
            
            # Could also:
            # - Send Zoom chat message
            # - Show overlay on screen
            # - Automatically pause share (if using SDK)
    
    def start_with_zoom(self):
        """Start monitoring with Zoom-specific features"""
        print("🎥 Zoom Monitor Integration")
        print("Starting monitoring for Zoom screen shares...\n")
        
        # Detect when Zoom starts
        while not self._is_zoom_running():
            print("⏳ Waiting for Zoom to start...")
            time.sleep(5)
        
        print("✅ Zoom detected")
        
        # Wait for screen share to start
        while not self._is_zoom_sharing():
            print("⏳ Waiting for screen share to start...")
            time.sleep(5)
        
        print("📹 Screen share detected - starting monitor\n")
        self.monitor.start_monitoring()
        
        # Monitor until share stops
        try:
            while self._is_zoom_sharing():
                time.sleep(5)
        except KeyboardInterrupt:
            pass
        
        print("\n📵 Screen share stopped")
        self.monitor.stop_monitoring()
    
    def _is_zoom_running(self) -> bool:
        """Check if Zoom is running"""
        detector = ScreenShareDetector()
        return detector.is_process_running(['zoom', 'zoom.us'])
    
    def _is_zoom_sharing(self) -> bool:
        """Check if Zoom is screen sharing"""
        detector = ScreenShareDetector()
        return detector._is_zoom_sharing()


# ========================================
# TEAMS INTEGRATION
# ========================================

class TeamsMonitorIntegration:
    """Integration for Microsoft Teams"""
    
    def __init__(self):
        self.monitor = RealTimeScreenShareMonitor(
            scan_interval=2.0,
            use_ml=True,
            alert_callback=self.teams_alert
        )
    
    def teams_alert(self, results):
        """Custom alert for Teams"""
        detections = results['detections']
        if not detections:
            return
        
        # Could integrate with Teams API to:
        # - Send a Teams message
        # - Create an incident in Teams
        # - Notify security team
        
        print("\n🔔 TEAMS ALERT - Sensitive content detected in share!")


# ========================================
# BROWSER-BASED (Meet, Zoom PWA, etc.)
# ========================================

class BrowserShareMonitor:
    """
    Monitor for browser-based screen sharing
    (Google Meet, Zoom PWA, etc.)
    """
    
    def __init__(self, browser: str = 'chrome'):
        self.browser = browser
        self.monitor = RealTimeScreenShareMonitor(
            scan_interval=1.5,  # Faster for browser shares
            use_ml=True
        )
    
    def is_browser_sharing(self) -> bool:
        """
        Detect if browser is screen sharing
        Checks for specific permissions/indicators
        """
        # Check if browser process exists with high CPU
        # (indicates active media capture)
        
        try:
            for proc in psutil.process_iter(['name', 'cpu_percent']):
                name = proc.info['name'].lower()
                if self.browser.lower() in name:
                    if proc.info['cpu_percent'] > 3.0:
                        return True
        except:
            pass
        
        return False


# ========================================
# USAGE EXAMPLES
# ========================================

def example_auto_monitor():
    """Example: Automatic monitoring"""
    from realtime_monitor import console_alert, desktop_notification_alert
    
    monitor = AutoScreenShareMonitor(
        scan_interval=2.0,
        use_ml=True,
        use_gpu=True,
        alert_callback=lambda r: (
            console_alert(r),
            desktop_notification_alert(r)
        )
    )
    
    monitor.run()


def example_zoom_specific():
    """Example: Zoom-specific monitoring"""
    zoom = ZoomMonitorIntegration()
    zoom.start_with_zoom()


def example_manual_trigger():
    """Example: Manual start/stop (hotkey triggered)"""
    from pynput import keyboard
    
    monitor = RealTimeScreenShareMonitor(
        scan_interval=2.0,
        use_ml=True
    )
    
    def on_press(key):
        try:
            if key == keyboard.Key.f9:  # F9 to start
                print("\n▶️  Starting monitor...")
                monitor.start_monitoring()
            elif key == keyboard.Key.f10:  # F10 to stop
                print("\n⏸️  Stopping monitor...")
                monitor.stop_monitoring()
        except:
            pass
    
    print("Press F9 to start monitoring, F10 to stop")
    
    with keyboard.Listener(on_press=on_press) as listener:
        listener.join()


def example_scheduled_monitoring():
    """Example: Monitor only during work hours"""
    from datetime import datetime
    
    monitor = RealTimeScreenShareMonitor(scan_interval=2.0)
    
    while True:
        now = datetime.now()
        
        # Only monitor 9 AM - 6 PM on weekdays
        is_work_hours = (
            now.weekday() < 5 and  # Monday-Friday
            9 <= now.hour < 18     # 9 AM - 6 PM
        )
        
        if is_work_hours:
            if not monitor.is_monitoring:
                print(f"⏰ Work hours - starting monitor")
                monitor.start_monitoring()
        else:
            if monitor.is_monitoring:
                print(f"🌙 Outside work hours - stopping monitor")
                monitor.stop_monitoring()
        
        time.sleep(60)  # Check every minute


if __name__ == "__main__":
    # Run auto monitor
    example_auto_monitor()
