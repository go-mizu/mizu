import 'package:flutter/foundation.dart';

/// Application state
class AppState extends ChangeNotifier {
  bool _isOnboarded = false;
  bool get isOnboarded => _isOnboarded;

  AppTab _selectedTab = AppTab.home;
  AppTab get selectedTab => _selectedTab;

  void completeOnboarding() {
    _isOnboarded = true;
    notifyListeners();
  }

  void selectTab(AppTab tab) {
    _selectedTab = tab;
    notifyListeners();
  }
}

enum AppTab {
  home,
  profile,
  settings,
}
