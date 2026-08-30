cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_pad;
};

struct VSInput
{
    float2 pos : POSITION;
    float2 st : TEXCOORD0;
    float4 content_mask : CONTENT_MASK;
    float4 color : COLOR;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float2 st : TEXCOORD0;
    float4 content_mask : CONTENT_MASK;
    float4 color : COLOR;
};

PSInput vs_main(VSInput input)
{
    PSInput output;
    float2 ndc = float2(
        (input.pos.x / u_viewport_size.x) * 2.0 - 1.0,
        1.0 - (input.pos.y / u_viewport_size.y) * 2.0
    );

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = input.pos;
    output.st = input.st;
    output.content_mask = input.content_mask;
    output.color = input.color;
    return output;
}

float4 ps_main(PSInput input) : SV_TARGET
{
    // Content mask clip
    if (input.content_mask.z > input.content_mask.x && input.content_mask.w > input.content_mask.y)
    {
        if (input.screen_pos.x < input.content_mask.x || input.screen_pos.x > input.content_mask.z ||
            input.screen_pos.y < input.content_mask.y || input.screen_pos.y > input.content_mask.w)
        {
            discard;
        }
    }

    float alpha = 1.0;
    // Quadratic bezier curve test
    if (input.st.x != 0.0 || input.st.y != 1.0)
    {
        float f = input.st.x * input.st.x - input.st.y;
        float2 grad = float2(ddx(f), ddy(f));
        float g_len = length(grad);
        if (g_len > 0.00001)
        {
            alpha = saturate(0.5 - f / g_len);
        }
        else
        {
            alpha = (f <= 0.0) ? 1.0 : 0.0;
        }
    }

    return input.color * alpha;
}
