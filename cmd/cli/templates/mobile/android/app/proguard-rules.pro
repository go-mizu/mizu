# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.
#
# For more details, see
#   http://developer.android.com/guide/developing/tools/proguard.html

# Keep Mizu runtime classes
-keep class * extends {{.Package}}.runtime.MizuError { *; }
-keep class {{.Package}}.runtime.APIError { *; }
-keep class {{.Package}}.runtime.AuthToken { *; }

# Keep SDK types for serialization
-keepclassmembers class {{.Package}}.sdk.** { *; }

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**
-keep class okhttp3.** { *; }

# Kotlin Serialization
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt

-keepclassmembers class kotlinx.serialization.json.** {
    *** Companion;
}
-keepclasseswithmembers class kotlinx.serialization.json.** {
    kotlinx.serialization.KSerializer serializer(...);
}

-keep,includedescriptorclasses class {{.Package}}.**$$serializer { *; }
-keepclassmembers class {{.Package}}.** {
    *** Companion;
}
-keepclasseswithmembers class {{.Package}}.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Keep generic signatures for deserialization
-keepattributes Signature
