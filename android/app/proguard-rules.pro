# Keep all gomobile-generated classes (wormhole AAR)
-keep class wormhole.** { *; }
-keep interface wormhole.** { *; }
-keepclassmembers class wormhole.** { *; }

# Keep Go/gomobile JNI entry points
-keep class go.** { *; }
-keep interface go.** { *; }

# Sentry
-keep class io.sentry.** { *; }
-dontwarn io.sentry.**

# Firebase
-keep class com.google.firebase.** { *; }
-keep class com.google.android.gms.** { *; }
-dontwarn com.google.firebase.**
-dontwarn com.google.android.gms.**
