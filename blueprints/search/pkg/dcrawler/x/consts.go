package x

// Search mode constants (replacing twitterscraper.SearchMode enum).
const (
	SearchTop    = "Top"
	SearchLatest = "Latest"
	SearchPhotos = "Photos"
	SearchVideos = "Videos"
	SearchPeople = "People"
)

const (
	bearerToken = "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	graphqlBaseURL = "https://x.com/i/api/graphql/"

	// GraphQL endpoint paths — query IDs extracted from x.com main.js bundle.
	// These rotate periodically; update by fetching x.com and parsing main.*.js.
	gqlUserByScreenName        = "-oaLodhGbbnzJBACb1kk2Q/UserByScreenName"
	gqlUserById                = "0aTrQMKgj95K791yXeNDRA/TweetResultByRestId" // also used for user by ID
	gqlUserTweetsV2            = "a3SQAz_VP9k8VWDr9bMcXQ/UserTweets"
	gqlUserTweetsAndRepliesV2  = "NullQbZlUJl-u6oBYRdrVw/UserTweetsAndReplies"
	gqlUserMedia               = "8HCIrWwy4C0fBTbPnMq5aA/UserMedia"
	gqlConversationTimeline    = "Kzfv17rukSzjT96BerOWZA/TweetDetail"
	gqlSearchTimeline          = "f_A-Gyo204PRxixpkrchJg/SearchTimeline"
	gqlFollowers               = "oQWxG6XdR5SPvMBsPiKUPQ/Followers"
	gqlFollowing               = "i2GOldCH2D3OUEhAdimLrA/Following"
	gqlRetweeters              = "X-XEqG5qHQSAwmvy00xfyQ/Retweeters" // may be removed from web
	gqlBookmarks               = "3aNu1FmuQHdPr0we_MsmmA/BookmarkSearchTimeline"
	gqlHomeTimeline            = "XzjVq_S9RnjdhmUGGPjpuw/HomeTimeline"
	gqlHomeLatestTimeline      = "ZibLTUqUvOqCmyVWrey-GA/HomeLatestTimeline"
	gqlListById                = "cIUpT1UjuGgl_oWiY7Snhg/ListByRestId"
	gqlListBySlug              = "K6wihoTiTrzNzSF8y1aeKQ/ListBySlug"
	gqlListMembers             = "fuVHh5-gFn8zDBBxb8wOMA/ListMembers"
	gqlListTweets              = "VQf8_XQynI3WzH6xopOMMQ/ListTimeline"
	gqlFavoriters              = "srMWv6gbkAGVm-0s2CVRlQ/Favoriters"
	gqlExplorePage             = "fIgAQhnH-MiqWGZ8YyIyJQ/ExplorePage"

	// Feature flags (from Nitter, minified).
	gqlFeatures = `{"android_ad_formats_media_component_render_overlay_enabled":false,"android_graphql_skip_api_media_color_palette":false,"android_professional_link_spotlight_display_enabled":false,"blue_business_profile_image_shape_enabled":false,"commerce_android_shop_module_enabled":false,"creator_subscriptions_subscription_count_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"freedom_of_speech_not_reach_fetch_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"hidden_profile_likes_enabled":false,"highlights_tweets_tab_ui_enabled":false,"interactive_text_enabled":false,"longform_notetweets_consumption_enabled":true,"longform_notetweets_inline_media_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_richtext_consumption_enabled":true,"mobile_app_spotlight_module_enabled":false,"responsive_web_edit_tweet_api_enabled":true,"responsive_web_enhance_cards_enabled":false,"responsive_web_graphql_exclude_directive_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_media_download_video_enabled":false,"responsive_web_text_conversations_enabled":false,"responsive_web_twitter_article_tweet_consumption_enabled":true,"unified_cards_destination_url_params_enabled":false,"responsive_web_twitter_blue_verified_badge_is_enabled":true,"rweb_lists_timeline_redesign_enabled":true,"spaces_2022_h2_clipping":true,"spaces_2022_h2_spaces_communities":true,"standardized_nudges_misinfo":true,"subscriptions_verification_info_enabled":true,"subscriptions_verification_info_reason_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"super_follow_badge_privacy_enabled":false,"super_follow_exclusive_tweet_notifications_enabled":false,"super_follow_tweet_api_enabled":false,"super_follow_user_api_enabled":false,"tweet_awards_web_tipping_enabled":false,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"tweetypie_unmention_optimization_enabled":false,"unified_cards_ad_metadata_container_dynamic_card_content_query_enabled":false,"verified_phone_label_enabled":false,"vibe_api_enabled":false,"view_counts_everywhere_api_enabled":true,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"responsive_web_jetfuel_frame":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_grok_imagine_annotation_enabled":true,"rweb_tipjar_consumption_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"creator_subscriptions_quote_tweet_preview_enabled":false,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_post_followups_enabled":true,"rweb_video_timestamps_enabled":false,"responsive_web_grok_share_attachment_enabled":true,"articles_preview_enabled":true,"immersive_video_status_linkable_timestamps":false,"articles_api_enabled":false,"responsive_web_grok_analysis_button_from_backend":true,"rweb_video_screen_enabled":false,"payments_enabled":false,"responsive_web_profile_redirect_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_community_note_auto_translation_is_enabled":false,"profile_label_improvements_pcf_label_in_profile_enabled":false,"grok_android_analyze_trend_fetch_enabled":false,"grok_translations_community_note_auto_translation_is_enabled":false,"grok_translations_post_auto_translation_is_enabled":false,"grok_translations_community_note_translation_is_enabled":false,"grok_translations_timeline_user_bio_auto_translation_is_enabled":false,"subscriptions_feature_can_gift_premium":false,"responsive_web_twitter_article_notes_tab_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":false,"hidden_profile_subscriptions_enabled":false,"responsive_web_grok_annotations_enabled":false,"post_ctas_fetch_enabled":false}`

	// Field toggles for different query types.
	userFieldToggles         = `{"withPayments":false,"withAuxiliaryUserLabels":true}`
	userTweetsFieldToggles   = `{"withArticlePlainText":false}`
	tweetDetailFieldToggles  = `{"withArticleRichContentState":true,"withArticlePlainText":false,"withGrokAnalyze":false,"withDisallowedReplyControls":false}`
)

// userAgent is the Chrome user-agent sent in all API requests.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
